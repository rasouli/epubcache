package server

import (
	"epubcache/attrs"
	obj "epubcache/objects"
	"epubcache/storage"
	"io/ioutil"
	"net/http"
	"path"

	log "github.com/sirupsen/logrus"
)

func NewShelfOueryHandlerV2(config *obj.Config, manager *obj.KeyManager, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.ShelfQueryMsgV2{}
		err := ReadMessage(req, &msg)

		if err != nil {

			bb, _ := ioutil.ReadAll(req.Body)

			logger.WithFields(log.Fields{"host": req.Host, "body": string(bb), "error": err}).Error("made invalid request to the shelf api.")
			WriteStatus(res, 400, false, "Bad Request Format.")
			return
		}

		refStore := storage.RefStore{config}

		shelfQueryResponse := obj.ShelfQueryResponseV2{}

		var hashFileList []string

		hashFileList = append(hashFileList, msg.Hashes...)
		hashFileList = append(hashFileList, msg.Files...)

		itemExists := func(slice []string, item string) bool {

			for _, val := range slice {
				if val == item {
					return true
				}
			}

			return false
		}

		for _, item := range hashFileList {

			belongsToFile := itemExists(msg.Files, item)
			belongsToHash := itemExists(msg.Hashes, item)

			curHash := ""

			if belongsToFile {
				fileAbsPath := path.Join(config.DrivePath, msg.User, item)
				curFileHash, err := refStore.ComputeHash(fileAbsPath)

				if err != nil {
					logger.WithFields(log.Fields{"file": item, "hash": curFileHash, "error": err, "abspath": fileAbsPath}).Error("unable to calculate the hash")
					continue
				}

				curHashExists := refStore.HashDirExists(curFileHash)
				if !curHashExists {
					continue
				}

				curHash = curFileHash
			}

			if belongsToHash {
				hashDirExists := refStore.HashDirExists(item)
				if !hashDirExists {
					continue
				}

				curHash = item
			}


			storePath := refStore.GetHashAbsDir(curHash)
			hasAttributeFile := attrs.AttributeFileExists(storePath, config.CacheSubDirName, config.CacheAttributeFileName)

			if !hasAttributeFile {
				continue
			}

			attributes, err := attrs.GetMediaAttributes(storePath, config.CacheSubDirName, config.CacheAttributeFileName)

			if err != nil {
				logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err, "mediaDirectory": storePath}).Error("Failed To Parse Attribute File")
				continue
			}

			hasCoverImage := attrs.HasCoverImage(attributes)

			if hasCoverImage {
				coverImagePath := attrs.GetCoverImageAbsPath(storePath, config.CacheSubDirName, attributes)
				base64Str, err := attrs.GetImageBase64(coverImagePath)
				if err != nil {
					logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err, "image": coverImagePath}).Error("Failed To Convert cover image to the base64")
					continue
				}
				attributes["_cover"] = base64Str
			} else {
				attributes["_cover"] = ""
			}

			// client can define which item this attribute belongs to
			attributes["_id"] = item

			if belongsToFile {
				shelfQueryResponse.Files = append(shelfQueryResponse.Files, attributes)
			} else if belongsToHash {
				shelfQueryResponse.Hashes = append(shelfQueryResponse.Hashes, attributes)
			}
		}


		WriteStatusWithObj(res, 200, shelfQueryResponse)
	})

}
