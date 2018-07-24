package server

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
	"fmt"
	obj "epubcache/objects"
	"epubcache/storage"
)



func NewTempLinkHandler(config *obj.Config, gen *obj.LinkGen, manager *obj.KeyManager ,logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.TempLinkMsg{}
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

		// check if the etag exists in the database

		key , err := manager.FindKey(msg.User, msg.Etag)

		if err != nil {
			WriteStatus(res, 404, false, "the expected etag does not reference any file.")
			logger.WithFields(log.Fields{"etag": msg.Etag, "user": msg.User, "host": req.Host, "error": err}).Error("the expected etag does not reference any file.")
			return
		}

		refStore := storage.RefStore{config}
		//storePath := extractor.GetStorePath(config.StoragePath, msg.User, msg.Etag)

		hashDirExists := refStore.HashDirExists(key.Hash)

		if !hashDirExists {
			WriteStatus(res, 404, false, "the hash assigned to this etag does not exists")
			logger.WithFields(log.Fields{"hash":key.Hash,"etag": msg.Etag, "user": msg.User, "host": req.Host, "error": err}).Error("the hash assigned to this etag does not exists")
			return
		}

		storePath := refStore.GetHashAbsDir(key.Hash)

		// check out for other search links
		searchLink := &obj.TempLink{}
		searchLink.Etag = msg.Etag
		searchLink.User = msg.User
		oldLink, err := gen.FetchTempLink(searchLink)

		var link *obj.TempLink
		if err != nil {
			logger.WithFields(log.Fields{"etag": msg.Etag, "user": msg.User, "host": req.Host, "error": err}).Error("error while searching for an old link")
			link = gen.CreateTempLink(msg.User, msg.Etag, time.Duration(config.TempLinkValidationDuration)*time.Minute, key.Hash)
			err = gen.PersistTempLink(link, storePath, config.ServeDirectory)

			if err != nil {
				WriteStatus(res, 500, false, "failed to persist the temp link")
				logger.WithFields(log.Fields{"host": req.Host, "user": msg.User, "etag": msg.Etag, "symlink": link.Link, "error": err}).Error("failed to persist link")
				return
			}

		} else {
			link = oldLink
		}

		cacheServeUrl := fmt.Sprintf("%s/%s",config.ServeUrl, link.Link)
		WriteStatus(res, 200, true, cacheServeUrl)
		logger.WithFields(log.Fields{"host": req.Host, "user": msg.User, "etag": msg.Etag, "link": cacheServeUrl, "symlink": link.Link, "hash": key.Hash}).Info("generated temp link.")

	})
}
