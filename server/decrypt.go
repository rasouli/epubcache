package server

import (
	extract "epubcache/extractor"
	"epubcache/keymanager"
	obj "epubcache/objects"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"epubcache/attrs"
	"epubcache/storage"
	"path"

)

func NewDecryptHandler(config *obj.Config, manager *obj.KeyManager, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.FileKeyMsg{}
		err := ReadMessage(req, &msg)

		if err != nil {

			bb, _ := ioutil.ReadAll(req.Body)

			logger.WithFields(log.Fields{"host": req.Host, "body": string(bb), "error": err}).Error("made invalid request to the decrypt api.")
			WriteStatus(res, 400, false, "Bad Request Format.")
			return
		}

		encryptedFileName := filepath.Base(msg.Path)
		hasPlcFormat := filepath.Ext(encryptedFileName) == obj.PLC_FORMAT
		encryptedFileFormat := filepath.Ext(encryptedFileName)
		decryptedFileFormat := encryptedFileFormat

		// in case of an .plc format, remove the plc format and get the last format before
		// the .plc
		if hasPlcFormat {
			decryptedFileFormat = filepath.Ext(encryptedFileName[:len(encryptedFileName)-len(obj.PLC_FORMAT)])
		}

		isEpub := decryptedFileFormat == obj.EPUB_FORMAT
		isPdf := decryptedFileFormat == obj.PDF_FORMAT
		decryptedFileName := fmt.Sprintf("%s%s", msg.Etag, decryptedFileFormat)

		encryptedFileSrcPath := filepath.Join(config.DrivePath, msg.User, msg.Path)
		decryptedFileDstPath := filepath.Join(config.TempStoragePath, decryptedFileName)

		err = keymanager.DecryptFile(encryptedFileSrcPath, decryptedFileDstPath, msg.Key)

		if err != nil {
			logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "host": req.Host, "message": msg, "error": err}).Error("There was an error with the decryption of the file")
			WriteStatus(res, 500, false, "Not Found.")
			return
		}

		refStore := storage.RefStore{config}
		hash, err := refStore.ComputeHash(decryptedFileDstPath)
		hashAlreadyExists := refStore.HashDirExists(hash)

		if !hashAlreadyExists {

			_, err := refStore.GetHashDirOrCreate(hash)
			storePath := refStore.GetHashAbsDir(hash)

			if err != nil {
				logger.WithFields(log.Fields{"hash": hash,"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "host": req.Host, "message": msg, "error": err}).Error("Failed To Compute/Create Hash Directory")
				WriteStatus(res, 500, false, "failed to compute/create hash directory")
				return
			}


			if isEpub {
				// if this file is an epub extract it
				err = extract.ExtractEpubContent(decryptedFileDstPath, storePath, config.OwnerUid, config.OwnerGid)
			} else {
				//otherwise just copy the file there
				fileCopyDestPath := filepath.Join(storePath, decryptedFileName)
				err = extract.CopyFile(decryptedFileDstPath, fileCopyDestPath)
			}

			if err != nil {
				logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "host": req.Host, "message": msg, "error": err}).Error("failed to create the extraction path")
				WriteStatus(res, 500, false, "failed to create the extraction path")
				return
			}

			// creating metadata directory under the extraction path
			//metaDir := extract.GetMetaDataDir(config.StoragePath, msg.User, msg.Etag, config.CacheSubDirName)
			metaDir := path.Join(storePath, config.CacheSubDirName)
			err = os.MkdirAll(metaDir, os.ModePerm)

			if err != nil {
				logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "host": req.Host, "message": msg, "error": err}).Error("failed to create the meta dir path")
				WriteStatus(res, 500, false, "failed to create the meta dir path")
				return
			}

			// process attributes and cover image. save cover image as _cover attribute in the attribute file
			attributes := make(map[string]string)
			if len(msg.Attributes) > 0 {
				attributes = msg.Attributes
			}

			if msg.CoverFile != "" {
				coverFileName, err := extract.CopyAndResizeCoverImage(metaDir, msg.CoverFile, config.CacheCoverFileName, config.CoverQuality, config.CoverSize)

				if err != nil {
					logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "coverFile": msg.CoverFile, "host": req.Host, "message": msg, "error": err}).Error("failed to save cover image file to meta directory.")
					WriteStatus(res, 500, false, "failed to save cover image file to meta directory.")
					return
				}

				attributes["_cover"] = coverFileName

			} else if isPdf {

				pdfCoverName := fmt.Sprintf("%s.jpg", config.CacheCoverFileName)
				err = attrs.RenderPDF(storePath, decryptedFileName, config.CacheSubDirName, pdfCoverName, config.CoverQuality, config.CoverSize)
				if err != nil {
					logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "coverFile": msg.CoverFile, "host": req.Host, "message": msg, "error": err}).Error("failed to render pdf first page image")
					WriteStatus(res, 500, false, "failed to render pdf file image.")
					return
				}

				attributes["_cover"] = pdfCoverName

			} else {
				attributes["_cover"] = ""
			}

			sizeInMb, err := extract.GetFileSizeInMB(decryptedFileDstPath)

			if err != nil {
				logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "host": req.Host, "message": msg, "error": err}).Error("failed to get the file size")
				WriteStatus(res, 500, false, "failed to get the file size")
			}

			attributes["_name"] = decryptedFileName
			attributes["_size"] = fmt.Sprintf("%.2f MB", sizeInMb)
			// save metadata file to the metadata dir
			err = extract.SaveMetaFile(metaDir, config.CacheAttributeFileName, attributes)

			if err != nil {
				logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "host": req.Host, "message": msg, "error": err}).Error("failed to save attributes to meta file.")
				WriteStatus(res, 500, false, "failed to save attributes to meta file.")
				return
			}

			// copy file for streaming to mobile clients
			err = extract.CopyFile(decryptedFileDstPath, path.Join(metaDir, decryptedFileName))
			if err != nil {
				logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "host": req.Host, "message": msg, "error": err}).Error("Error While Copying instance of the decrypted file.")
				WriteStatus(res, 500, false, "failed to copy instance of the file.")
				return
			}

			// chown the meta directory
			err = extract.ChownR(metaDir, config.OwnerUid, config.OwnerGid)

			if err != nil {
				logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "epubExtractPath": storePath, "metadir": metaDir, "host": req.Host, "message": msg, "error": err}).Error("Error While ChownR Meta  Directory")
				WriteStatus(res, 500, false, "failed to chown the meta directory.")
				return
			}

		} // hash already exists. ignore processing file.

		// persist decryption , etag , user, hash to db
		err = manager.PersistKey(&obj.FileKey{Key: msg.Key, Etag: msg.Etag, User: msg.User, Hash: hash})
		if err != nil {
			logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "host": req.Host, "message": msg, "error": err}).Error("Error While Persisting FileKey")
			WriteStatus(res, 500, false, "error while persisting key.")
			return
		}

		// remove files
		err = os.Remove(decryptedFileDstPath)


		if err != nil {
			logger.WithFields(log.Fields{"encryptedFileSrc": encryptedFileSrcPath, "decryptedFileDstPath": decryptedFileDstPath, "host": req.Host, "message": msg, "error": err}).Error("Error While Removing Decrypted  File")
			WriteStatus(res, 500, false, "error while persisting key.")
		} else {
			WriteStatus(res, 200, true, "ok.")
		}
	})
}
