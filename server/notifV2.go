package server

import (
	"encoding/json"
	extract "epubcache/extractor"
	obj "epubcache/objects"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"epubcache/attrs"
	"epubcache/epub"
	"epubcache/storage"
	"fmt"
	"path"
	"path/filepath"
	"github.com/satori/go.uuid"
)

// notify api: get the specified file

func NewNotifHandlerV2(config *obj.Config, notifLogger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		if req.Method != "POST" {
			return
		}

		notif := obj.NotifyMsg{}
		err := json.NewDecoder(req.Body).Decode(&notif)

		if err != nil {

			bodyStr, _ := ioutil.ReadAll(req.Body)
			notifLogger.WithFields(log.Fields{"host": req.Host, "body": bodyStr, "error": err}).Error("Invalid Json Request Body")
			WriteStatus(res, 400, false, "could not parse json notify request.")
			return
		}

		refStore := storage.RefStore{config}


		uploadedFileAbsPath := filepath.Join(config.DrivePath, notif.User, notif.Path)
		uploadedFileName := filepath.Base(uploadedFileAbsPath)
		upladedFileFormat := filepath.Ext(uploadedFileName)

		uid, _ := uuid.NewV4()
		canonicalName := fmt.Sprintf("%s%s", uid.String(), upladedFileFormat)


		uploadedFileIsPlc := obj.PLC_FORMAT == upladedFileFormat
		uploadedFileIsPdf := obj.PDF_FORMAT == upladedFileFormat
		uploadedFileIsEpub := obj.EPUB_FORMAT == upladedFileFormat

		hash, err := refStore.ComputeHash(uploadedFileAbsPath)
		hashAlreadyExists := refStore.HashDirExists(hash)
		storePath := refStore.GetHashAbsDir(hash)
		if !hashAlreadyExists {
			_, err := refStore.GetHashDirOrCreate(hash)

			if err != nil {
				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err, "hash": hash, "storePath": storePath}).Error(" Error while creating reference storage")
				WriteStatus(res, 400, false, "failed to create  reference path")
				return
			}

			metaDir := path.Join(storePath, config.CacheSubDirName)

			if uploadedFileIsPlc {

				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err}).Error("Got .PLC file format. this api does not support decryption.")
				WriteStatus(res, 400, false, "file in .PLC format is not supported.")
				return
			}

			// create METADIR and all of it's ancestors
			err = os.MkdirAll(metaDir, os.ModePerm)

			if err != nil {
				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err}).Error("Error While Creating Metadata Directory")
				WriteStatus(res, 400, false, "failed to create metadata directory path")
				return
			}

			attributes := map[string]string{}

			if uploadedFileIsEpub {

				epubExtractPath := storePath

				book, err := epub.Open(uploadedFileAbsPath)

				if err != nil {
					notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err}).Error("Error While Reading Epub File")
					WriteStatus(res, 400, false, "failed to open uploaded epub file")
					return
				}

				err = extract.ExtractEpubContent(uploadedFileAbsPath, epubExtractPath, config.OwnerUid, config.OwnerGid)

				if err != nil {
					notifLogger.WithFields(log.Fields{"epubFile": uploadedFileAbsPath, "extractPath": epubExtractPath, "error": err}).Error("Error While Extracting Epub File")
					WriteStatus(res, 400, false, "error while extracting epub content.")
					return
				}

				attributes = attrs.GetEpubMetadataFromFile(book, uploadedFileAbsPath)
				coverImagePath, _ := attrs.GetCoverImageFromXHtml(book, epubExtractPath)

				if coverImagePath == "" {
					attributes["_cover"] = ""
				} else {

					coverFileName, err := extract.CopyAndResizeCoverImage(metaDir, coverImagePath, config.CacheCoverFileName, config.CoverQuality, config.CoverSize)
					if err != nil {
						notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "coverImagePath": coverImagePath, "error": err}).Error("Error While Copying Epub Cover File")
						WriteStatus(res, 400, false, "Failed To Extract Epub Cover File.")
						return
					}

					attributes["_cover"] = coverFileName
				}

			} else if uploadedFileIsPdf {
				//pdfFileName := uploadedFileName
				pdfDestFile := filepath.Join(storePath, canonicalName)
				err = extract.CopyFile(uploadedFileAbsPath, pdfDestFile)

				if err != nil {
					notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "pdfDestFile": pdfDestFile, "error": err}).Error("Error While Copying Pdf File")
					WriteStatus(res, 400, false, "Failed To Copy PDF file.")
					return
				}

				pdfCoverName := fmt.Sprintf("%s.jpg", config.CacheCoverFileName)
				err = attrs.RenderPDF(storePath, canonicalName, config.CacheSubDirName, pdfCoverName, config.CoverQuality, config.CoverSize)
				if err != nil {
					notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "pdfDestFile": pdfDestFile, "error": err}).Error("Error While Rendering Pdf File")
					WriteStatus(res, 400, false, "Failed To Copy PDF file.")
					return
				}

				attributes["_cover"] = pdfCoverName
			}

			sizeInMb, err := extract.GetFileSizeInMB(uploadedFileAbsPath)

			if err != nil {
				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err}).Error("Error while getting uploaded file size.")
				WriteStatus(res, 400, false, "error while getting the size of the file")
				return
			}

			attributes["_size"] = fmt.Sprintf("%.2f MB", sizeInMb)
			attributes["_name"] = canonicalName

			err = extract.SaveMetaFile(metaDir, config.CacheAttributeFileName, attributes)

			if err != nil {
				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err, "metaDir": metaDir}).Error("Could not save metadata file.")
				WriteStatus(res, 400, false, "error while saving metadata.")
				return
			}

			err = extract.CopyFile(uploadedFileAbsPath, path.Join(metaDir, canonicalName))

			if err != nil {
				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err, "metaDir": metaDir}).Error("Error while copying uploaded file to ref directory.")
				WriteStatus(res, 400, false, "Error while copying instance of the file.")
				return
			}
			// chown the meta directory
			err = extract.ChownR(storePath, config.OwnerUid, config.OwnerGid)

			if err != nil {
				notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err, "metaDir": metaDir}).Error("Could not change permission.")
				WriteStatus(res, 400, false, "could not change the permission.")
				return
			}

		}

		//err = manager.PersistKey(&obj.FileKey{Key: "", Etag: notif.Etag, User: notif.User, Hash:hash})

		if err != nil {
			notifLogger.WithFields(log.Fields{"uploadedFile": uploadedFileAbsPath, "error": err}).Error("Could not persist file data to database.")
			WriteStatus(res, 400, false, "Could not persist file data to database.")
			return
		}

		WriteStatus(res, 200, true, hash)

	})
}
