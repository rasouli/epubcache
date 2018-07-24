package server

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"fmt"
	obj "epubcache/objects"
	"epubcache/storage"
	"epubcache/attrs"
	"path"
	"epubcache/keymanager"
	"github.com/satori/go.uuid"
)

func NewEncryptHandler(config *obj.Config, manager *obj.KeyManager, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.EncryptMsg{}
		err := ReadMessage(req, &msg)

		if err != nil {
			WriteStatus(res, 400, false, "Bad Request Format.")
			bb, err := ioutil.ReadAll(req.Body)

			if err != nil {
				logger.WithFields(log.Fields{"host": req.Host, "error": err}).Error("Bad Request format")
			}

			logger.WithFields(log.Fields{"host": req.Host, "body": string(bb), "error": err}).Error("made invalid request to the temp link api.")
			return
		}

		// check if the hash exists in the database

		key := msg.Key

		refStore := storage.RefStore{config}
		//storePath := extractor.GetStorePath(config.StoragePath, msg.User, msg.Etag)

		hashDirExists := refStore.HashDirExists(msg.Hash)

		if !hashDirExists {
			WriteStatus(res, 404, false, "this hash does not exists")
			logger.WithFields(log.Fields{"hash": msg.Hash, "key": msg.Key, "host": req.Host, "error": err}).Error("this hash does not exists")
			return
		}

		storePath := refStore.GetHashAbsDir(msg.Hash)
		attributes, err := attrs.GetMediaAttributes(storePath, config.CacheSubDirName, config.CacheAttributeFileName)

		if err != nil {
			WriteStatus(res, 500, false, "could not read from metadata file")
			logger.WithFields(log.Fields{"hash": msg.Hash, "key": msg.Key, "host": req.Host, "error": err}).Error("Could Not Read From Meta Data File")
			return
		}

		fileName, ok := attributes["_name"]

		if !ok {
			WriteStatus(res, 404, false, "the _file attribute was not found in the metadata file")
			logger.WithFields(log.Fields{"hash": msg.Hash, "key": msg.Key,  "host": req.Host, "error": err}).Error("the _file attribute was not found in the metadata file")
			return
		}

		sourceFile := path.Join(storePath, config.CacheSubDirName, fileName)

		encryptFormat := ""
		if key != "" {
			encryptFormat = ".plc"
		}
		uid, _ := uuid.NewV4()

		tempFile := path.Join(config.TempStoragePath, fmt.Sprintf("%s-%s%s", uid.String(), fileName, encryptFormat))

		err = keymanager.EncryptFile(sourceFile, tempFile, key)

		if err != nil {
			WriteStatus(res, 500, false, "Encryption failed.")
			logger.WithFields(log.Fields{"hash": msg.Hash, "key": msg.Key, "host": req.Host, "error": err}).Error("the encryption has failed.")
			return
		}


		WriteStatus(res, 200, true, tempFile)
	})
}
