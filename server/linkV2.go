package server

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
	"fmt"
	obj "epubcache/objects"
	"epubcache/storage"
	"path"
	"epubcache/attrs"
)

func NewTempLinkHandlerV2(config *obj.Config, gen *obj.LinkGen, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.TempLinkMsgV2{}
		err := ReadMessage(req, &msg)

		if err != nil {
			WriteStatus(res, 400, false, "Bad Request Format.")
			bb, err := ioutil.ReadAll(req.Body)

			if err != nil {
				logger.WithFields(log.Fields{"host": req.Host, "error": err}).Error("made request to the temp link api and failed to parse the body of the request to log.")
			}

			logger.WithFields(log.Fields{"host": req.Host, "body": string(bb), "error": err}).Error("made invalid request to the temp link api.")
			return
		}

		refStore := storage.RefStore{config}
		requestedHash := msg.Hash
		requestedFile := msg.File
		shouldComputeHash := requestedHash == "" && requestedFile != ""

		requestedFileHash := ""
		fileAbsPath := path.Join(config.DrivePath, msg.User, msg.File)

		if shouldComputeHash {

			requestedFileHash, err = refStore.ComputeHash(fileAbsPath)

			if err != nil {

				logger.WithFields(log.Fields{"file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath}).Error("could not compute requested file hash")
				WriteStatus(res, 500, false, "could not compute file hash")
			}
		} else {
			requestedFileHash = requestedHash
		}

		hashDirExists := refStore.HashDirExists(requestedFileHash)

		if !hashDirExists {
			logger.WithFields(log.Fields{"file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath, "computedHash": requestedFileHash}).Error("computed/requested hash does not exists")
			WriteStatus(res, 500, false, "hash not found")
		}

		hashAbsDirectory := refStore.GetHashAbsDir(requestedFileHash)

		attributes, err := attrs.GetMediaAttributes(hashAbsDirectory, config.CacheSubDirName, config.CacheAttributeFileName)

		if err != nil {
			logger.WithFields(log.Fields{"file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath, "computedHash": requestedFileHash}).Error("could not read the meta file")
			WriteStatus(res, 500, false, "could not read meta file")
		}

		fileName, ok := attributes["_name"]

		if !ok {
			logger.WithFields(log.Fields{"attributes": attributes, "file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath, "computedHash": requestedFileHash}).Error("_name key attribute not found")
			WriteStatus(res, 500, false, "_name attribute not found")
		}

		fileExtension := path.Ext(fileName)

		isPdf := fileExtension == obj.PDF_FORMAT

		searchLink := &obj.TempLink{}
		searchLink.User = msg.User
		searchLink.Hash = requestedFileHash
		oldLink, err := gen.FetchTempLink(searchLink)

		var link *obj.TempLink
		if err != nil {
			logger.WithFields(log.Fields{"attributes": attributes, "file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath, "computedHash": requestedFileHash}).Error("error while searching for an old link")
			link = gen.CreateTempLink(msg.User, "", time.Duration(config.TempLinkValidationDuration)*time.Minute, requestedFileHash)
			err = gen.PersistTempLink(link, hashAbsDirectory, config.ServeDirectory)

			if err != nil {
				WriteStatus(res, 500, false, "failed to persist the temp link")
				logger.WithFields(log.Fields{"attributes": attributes, "file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath, "computedHash": requestedFileHash}).Error("failed to persist link")
				return
			}

		} else {
			link = oldLink
		}

		cacheServeUrl := fmt.Sprintf("%s/%s", config.ServeUrl, link.Link)

		if isPdf {
			cacheServeUrl = fmt.Sprintf("%s/%s", cacheServeUrl, fileName)
		}
		WriteStatus(res, 200, true, cacheServeUrl)
		logger.WithFields(log.Fields{"hashAbsDirectory": hashAbsDirectory,"attributes": attributes, "file": msg.File, "hash": msg.Hash, "error": err, "fileAbsPath": fileAbsPath, "computedHash": requestedFileHash, "link": cacheServeUrl, "symlink": link.Link}).Info("generated temp link.")

	})
}
