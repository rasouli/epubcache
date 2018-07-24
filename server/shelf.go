package server

import (
	obj "epubcache/objects"
	"net/http"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	attrs "epubcache/attrs"
	"epubcache/storage"
)

func NewShelfOueryHandler(config *obj.Config, manager *obj.KeyManager, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.ShelfQueryMsg{}
		err := ReadMessage(req, &msg)

		if err != nil {

			bb, _ := ioutil.ReadAll(req.Body)

			logger.WithFields(log.Fields{"host": req.Host, "body": string(bb), "error": err}).Error("made invalid request to the decrypt api.")
			WriteStatus(res, 400, false, "Bad Request Format.")
			return
		}

		userKeys, err := manager.FindUserKeysByEtag(msg.User, msg.Etags)

		if err != nil {
			logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err}).Error("Failed To Get All User Files and Keys")
			WriteStatus(res, 500, false, "Failed To Query User Files ")
			return
		}


		shelfQueryResponse := obj.ShelfQueryResponse{}


		refStore := storage.RefStore{config}
		for _, k := range userKeys {
			storePath := refStore.GetHashAbsDir(k.Hash)
			hasAttributeFile := attrs.AttributeFileExists(storePath,config.CacheSubDirName,config.CacheAttributeFileName)

			if !hasAttributeFile{
				continue
			}


			attributes, err := attrs.GetMediaAttributes(storePath,config.CacheSubDirName,config.CacheAttributeFileName)

			if err != nil {
				logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err, "mediaDirectory": storePath}).Error("Failed To Parse Attribute File")
				continue
			}

			hasCoverImage := attrs.HasCoverImage(attributes)

			if hasCoverImage {
				coverImagePath := attrs.GetCoverImageAbsPath(storePath,config.CacheSubDirName,attributes)
				base64Str,err := attrs.GetImageBase64(coverImagePath)
				if err != nil {
					logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err, "image": coverImagePath}).Error("Failed To Convert cover image to the base64")
					continue
				}
				attributes["_cover"] = base64Str
			} else {
				attributes["_cover"] = ""
			}


			attributes["_etag"] = k.Etag
			shelfQueryResponse.Files = append(shelfQueryResponse.Files, attributes)


		}

		shelfQueryResponse.Message = "ok."
		shelfQueryResponse.Result ="success"

		WriteStatusWithObj(res,200,shelfQueryResponse)
	})
}
