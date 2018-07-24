package server

import (
	log "github.com/sirupsen/logrus"

	obj "epubcache/objects"
	"net/http"
	"io/ioutil"

)

func NewDeleteHandler(config *obj.Config, manager *obj.KeyManager, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		if req.Method != "POST" {
			WriteStatus(res, 400, false, "Bad Request.")
			return
		}

		msg := obj.DeleteMsg{}
		err := ReadMessage(req, &msg)

		if err != nil {

			bb, _ := ioutil.ReadAll(req.Body)

			logger.WithFields(log.Fields{"host": req.Host, "body": string(bb), "error": err}).Error("made invalid request to the decrypt api.")
			WriteStatus(res, 400, false, "Bad Request Format.")
			return
		}

		fileKey, err := manager.FindKey(msg.User, msg.Etag)

		if err != nil {
			logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err}).Error("No record found for the file")
			WriteStatus(res, 500, false, "No record found for the file ")
			return
		}


		err = manager.DeleteFileKey(fileKey)

		if err != nil {
			logger.WithFields(log.Fields{"host": req.Host, "message": msg, "error": err,}).Error("could not delete the file record from database")
			WriteStatus(res, 500, false, "Failed to remove the file record  from database ")
			return
		}


		WriteStatus(res,200,true, "ok.")
	})
}
