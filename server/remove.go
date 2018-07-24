package server

import (
	log "github.com/sirupsen/logrus"
	"os"
	obj "epubcache/objects"
	"path/filepath"
)

func RemoveOldSymlinks(config *obj.Config, gen *obj.LinkGen, logger *log.Logger) {
	serveDir := config.ServeDirectory
	links, err := gen.GetExpierdLinks()

	logger.Info("begining removing symlinks")
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Error("failed to get  expired links")
		return
	}

	for _, link := range links {
		linkPath := link.Link
		oldPath := filepath.Join(serveDir, linkPath)
		err = os.Remove(oldPath)

		if err != nil {
			logger.WithFields(log.Fields{"error": err, "link": link.Link, "expiry": link.Expiry, "symlink": oldPath}).Error("failed to remove symlink")
		}

		err = gen.Remove(&link)
		if err != nil {
			logger.WithFields(log.Fields{"link": link.Link, "expiry": link.Expiry, "user": link.User, "etag": link.Etag, "symlink": oldPath}).Error("failed to remove the symlink from database")
		} else {
			logger.WithFields(log.Fields{"link": link.Link, "expiry": link.Expiry, "user": link.User, "etag": link.Etag, "symlink": oldPath}).Info("removed symlink")
		}

	}
}
