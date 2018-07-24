package server

import (
	"encoding/json"
	obj "epubcache/objects"
	"fmt"
	"net/http"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"

	"epubcache/keymanager"
	"epubcache/linkgen"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
)

var wg sync.WaitGroup
var notifLogger *log.Logger
var notifLoggerV2 *log.Logger
var linkGenLogger *log.Logger
var linkLoggerV2 *log.Logger
var removeLogger *log.Logger
var decryptLogger *log.Logger
var decryptLoggerV2 *log.Logger
var encryptLogger *log.Logger
var shelfLogger *log.Logger
var shelfLoggerV2 *log.Logger
var deleteLogger *log.Logger
var notifServer *http.ServeMux

func LoadConfig() (obj.Config, error) {

	// if len(os.Args) == 2 {
	// 	config, err := LoadConfigFromFile(os.Args[1])
	// 	return config, err
	//
	// } else {
	config, err := LoadConfigFromEnv()

	if err != nil {
		fmt.Printf("Error: Failed To Read From Environment. error %s", err)
		fmt.Printf("Error: Trying To Read From Config File...")
		config, err = LoadConfigFromFile(os.Args[1])
	}

	return config, err

	// }

}

func LoadConfigFromFile(configPath string) (obj.Config, error) {
	config := obj.Config{}
	file, err := os.OpenFile(configPath, os.O_RDONLY, 0666)
	defer file.Close()
	err = json.NewDecoder(file).Decode(&config)

	if err != nil {
		return config, err
	}

	return config, nil
}

func LoadConfigFromEnv() (obj.Config, error) {
	config := obj.Config{}
	db := obj.DBConfiguration{}
	config.Db = db

	fromEnv := func(name string, defaultValue string) string {
		val, exists := os.LookupEnv(name)
		if !exists {
			return defaultValue
		}

		return val
	}

	// first check if os env flag is explicitly set
	cacheEnabled := fromEnv("EPUB_CACHE_ENV", "0")
	if cacheEnabled != "1" {

		return config, errors.New("Reading From Environment Variable Was Disabled")
	}

	config.Db.Database = fromEnv("EPUB_CACHE_CONFIG_DB_DATABASE", "")
	config.Db.Host = fromEnv("EPUB_CACHE_CONFIG_DB_HOST", "")
	config.Db.Password = fromEnv("EPUB_CACHE_CONFIG_DB_PASSWORD", "")

	portStr := fromEnv("EPUB_CACHE_CONFIG_DB_PORT", "")
	portNum, _ := strconv.Atoi(portStr)
	config.Db.Port = portNum

	config.Db.User = fromEnv("EPUB_CACHE_CONFIG_DB_USER", "")

	config.LogPath = fromEnv("EPUB_CACHE_CONFIG_LOG_PATH", "")
	config.DrivePath = fromEnv("EPUB_CACHE_CONFIG_DRIVE_FILES_ROOT_PATH", "")
	config.StoragePath = fromEnv("EPUB_CACHE_CONFIG_STORAGE_PATH", "")
	config.TempStoragePath = fromEnv("EPUB_CACHE_CONFIG_TEMP_STORAGE_PATH", "")
	config.StoragePermission = fromEnv("EPUB_CACHE_CONFIG_STORAGE_PERMISSION", "")
	config.NotifyServerPath = fromEnv("EPUB_CACHE_CONFIG_NOTIFY_SERVER", "")

	durationStr := fromEnv("EPUB_CACHE_CONFIG_LINK_DURATION", "")
	durationNum, _ := strconv.Atoi(durationStr)
	config.TempLinkValidationDuration = durationNum

	config.ServeDirectory = fromEnv("EPUB_CACHE_CONFIG_SERVE_DIRECTORY", "")
	config.ServeUrl = fromEnv("EPUB_CACHE_CONFIG_SERVE_URL", "")

	ownerGidStr := fromEnv("EPUB_CACHE_CONFIG_OWNER_GID", "")
	ownerGidNum, _ := strconv.Atoi(ownerGidStr)
	config.OwnerGid = ownerGidNum

	ownerUidStr := fromEnv("EPUB_CACHE_CONFIG_OWNER_UID", "")
	ownerUidNum, _ := strconv.Atoi(ownerUidStr)
	config.OwnerUid = ownerUidNum

	removeIntervalStr := fromEnv("EPUB_CACHE_CONFIG_REMOVE_INTERVAL", "")
	removeIntervalNum, _ := strconv.Atoi(removeIntervalStr)
	config.RemoveInterval = removeIntervalNum

	cacheAttributeFileNameStr := fromEnv("EPUB_CACHE_CONFIG_CACHE_ATTRIBUTE_FILE_NAME", "")
	config.CacheAttributeFileName = cacheAttributeFileNameStr

	cacheCoverFileNameStr := fromEnv("EPUB_CACHE_CONFIG_CACHE_COVER_FILE_NAME", "")
	config.CacheCoverFileName = cacheCoverFileNameStr

	cacheSubDirNameStr := fromEnv("EPUB_CACHE_CONFIG_CACHE_SUB_DIR_NAME", "")
	config.CacheSubDirName = cacheSubDirNameStr

	coverSize := fromEnv("EPUB_CACHE_CONFIG_COVER_SIZE", "")
	config.CoverSize = coverSize

	coverQuality := fromEnv("EPUB_CACHE_COVER_QUALITY", "")
	config.CoverQuality = coverQuality

	refrenceStore := fromEnv("EPUB_CACHE_CONFIG_REFERENCE_STORAGE_PATH", "")
	config.ReferenceStoragePath = refrenceStore

	hashLenStr := fromEnv("EPUB_CACHE_CONFIG_FILE_HASH_LEN", "")
	fileHashlenInt, _ := strconv.Atoi(hashLenStr)
	config.FileHashLen = fileHashlenInt

	return config, nil
}

func SetupLoggers(config obj.Config) error {

	notifLogger = log.New()
	file, err := os.OpenFile(filepath.Join(config.LogPath, "notify-api.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	notifLogger.Out = file
	notifLogger.Info("Initialized Logger.")

	linkGenLogger = log.New()
	file2, err := os.OpenFile(filepath.Join(config.LogPath, "link-api.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	linkGenLogger.Out = file2
	linkGenLogger.Info("Initialized Logger.")

	removeLogger = log.New()
	file3, err := os.OpenFile(filepath.Join(config.LogPath, "sheduleRemove.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	removeLogger.Out = file3
	removeLogger.Info("Initialized Logger.")

	decryptLogger = log.New()
	file4, err := os.OpenFile(filepath.Join(config.LogPath, "decrypt.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	decryptLogger.Out = file4
	decryptLogger.Info("Initialized Logger.")

	shelfLogger = log.New()
	file5, err := os.OpenFile(filepath.Join(config.LogPath, "shelf.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}

	shelfLogger.Out = file5
	shelfLogger.Info("Initialized Logger.")

	deleteLogger = log.New()
	file6, err := os.OpenFile(filepath.Join(config.LogPath, "delete.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}

	deleteLogger.Out = file6
	deleteLogger.Info("Initialized Logger.")

	//
	encryptLogger = log.New()
	file7, err := os.OpenFile(filepath.Join(config.LogPath, "encrypt.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}

	encryptLogger.Out = file7
	encryptLogger.Info("Initialized Logger.")

	decryptLoggerV2 = log.New()
	file8, err := os.OpenFile(filepath.Join(config.LogPath, "decryptV2.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	decryptLoggerV2.Out = file8
	decryptLoggerV2.Info("Initialized Logger.")

	notifLoggerV2 = log.New()
	file9, err := os.OpenFile(filepath.Join(config.LogPath, "notifV2.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	notifLoggerV2.Out = file9
	notifLoggerV2.Info("Initialized Logger.")

	linkLoggerV2 = log.New()
	file10, err := os.OpenFile(filepath.Join(config.LogPath, "linkV2.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	linkLoggerV2.Out = file10
	linkLoggerV2.Info("Initialized Logger.")

	shelfLoggerV2 = log.New()
	file11, err := os.OpenFile(filepath.Join(config.LogPath, "shelfV2.log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	shelfLoggerV2.Out = file11
	shelfLoggerV2.Info("Initialized Logger.")
	return nil
}

func SetupEncryptHandler(server *http.ServeMux, config *obj.Config, notifLogger *log.Logger) {
	manager, err := keymanager.New(config.Db)

	if err != nil {
		log.Error("could not initalize manager with the db configuration for  Encrypt Handler.", err)
		return
	}

	// setup notify handler
	server.Handle("/v2/encrypt", NewEncryptHandler(config, manager, notifLogger))
}

func SetupNotifHandler(server *http.ServeMux, config *obj.Config, notifLogger *log.Logger, notifLoggerV2 *log.Logger) {

	manager, err := keymanager.New(config.Db)
	if err != nil {
		log.Error("could not initalize manager with the db configuration for  Notif Handler.", err)
		return
	}
	// setup notify handler
	server.Handle("/notify", NewNotifHandler(config, manager, notifLogger))
	server.Handle("/v2/notify", NewNotifHandlerV2(config, notifLoggerV2))

}

func SetupLinkGenHandler(server *http.ServeMux, config *obj.Config, linkLogger *log.Logger, linkLogger2 *log.Logger) error {
	gen, err := linkgen.New(config.Db)
	if err != nil {
		log.Error("could not initalize link generator with the db configuration.", err)
		return err
	}

	manager, err := keymanager.New(config.Db)
	if err != nil {
		log.Error("could not initalize key manager with the db configuration.", err)
		return err
	}

	server.Handle("/link", NewTempLinkHandler(config, gen, manager, linkLogger))
	server.Handle("/v2/link", NewTempLinkHandlerV2(config, gen, linkLogger2))
	return nil
}

func SetupDecryptHandler(server *http.ServeMux, config *obj.Config, decLogger *log.Logger, decLoggerV2 *log.Logger) error {
	manager, err := keymanager.New(config.Db)
	if err != nil {
		log.Error("could not initalize manager with the db configuration.", err)
		return err
	}

	server.Handle("/decrypt", NewDecryptHandler(config, manager, decLogger))
	server.Handle("/v2/decrypt", NewDecryptHandlerV2(config, decLoggerV2))
	return nil
}

func SetupShelfQueryHandler(server *http.ServeMux, config *obj.Config, shelfLogger *log.Logger, shelfLoggerV2 *log.Logger) error {
	manager, err := keymanager.New(config.Db)
	if err != nil {
		log.Error("could not initialize manager with the db configuration for the shelf api", err)
		return err
	}

	server.Handle("/shelf", NewShelfOueryHandler(config, manager, shelfLogger))
	server.Handle("/v2/shelf", NewShelfOueryHandlerV2(config, manager, shelfLoggerV2))

	return nil
}

func SetupDeleteHandler(server *http.ServeMux, config *obj.Config, deleteLogger *log.Logger) error {
	manager, err := keymanager.New(config.Db)
	if err != nil {
		log.Error("could not initialize manager with db configuration for the delete api ")
		return err
	}

	server.Handle("/delete", NewDeleteHandler(config, manager, deleteLogger))
	return nil
}

func MigrateDatabase(config *obj.Config) error {
	dbLink := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true",
		config.Db.User,
		config.Db.Password,
		config.Db.Host,
		config.Db.Port,
		config.Db.Database)

	db, err := gorm.Open("mysql", dbLink)

	defer db.Close()

	if err != nil {
		return err
	}

	err = db.AutoMigrate(&obj.FileKey{}, &obj.TempLink{}).Error

	return err

}

func WriteStatus(res http.ResponseWriter, status int, success bool, message string) {
	res.WriteHeader(status)
	result := "error"
	if success {
		result = "success"
	}

	json.NewEncoder(res).Encode(map[string]string{"result": result, "message": message})
}

func WriteStatusWithObj(res http.ResponseWriter, status int, message interface{}) {
	res.WriteHeader(status)

	json.NewEncoder(res).Encode(message)
}

func ReadMessage(req *http.Request, obj interface{}) error {
	err := json.NewDecoder(req.Body).Decode(&obj)
	return err
}

func Serve() {
	wg.Add(1)

	config, err := LoadConfig()

	if err != nil {
		fmt.Printf("Error: Failed To Read From Config File/ENV: %s", err)
		return
	}

	fmt.Printf("loaded configuration: %+v", config)

	err = SetupLoggers(config)

	if err != nil {
		fmt.Printf("Error: Setting Up Log File Failed In Path: %s \n Reason: %s", config.LogPath, err)
		return
	}

	notifServer = http.NewServeMux()

	MigrateDatabase(&config)
	SetupNotifHandler(notifServer, &config, notifLogger, notifLoggerV2)
	SetupLinkGenHandler(notifServer, &config, linkGenLogger, linkLoggerV2)
	SetupDecryptHandler(notifServer, &config, decryptLogger, decryptLoggerV2)
	SetupShelfQueryHandler(notifServer, &config, shelfLogger, shelfLoggerV2)
	SetupDeleteHandler(notifServer, &config, deleteLogger)
	SetupEncryptHandler(notifServer, &config, encryptLogger)

	// the live controller for health check
	notifServer.Handle("/live", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		WriteStatus(res, 200, true, "Cache Live")
	}))

	go func() {
		http.ListenAndServe(config.NotifyServerPath, notifServer)
	}()

	ticker := time.NewTicker(time.Duration(config.RemoveInterval) * time.Minute)
	gen, err := linkgen.New(config.Db)

	if err != nil {
		fmt.Printf("Error: Setting Up Log File Failed In Path: %s \n Reason: %s", config.LogPath, err)
		return
	}

	go func() {
		for range ticker.C {
			RemoveOldSymlinks(&config, gen, removeLogger)
		}
	}()
	wg.Wait()

}
