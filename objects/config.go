package objects

type Config struct {
	LogPath                    string          `json:"logPath"`
	DrivePath                  string          `json:"driveFilesRootPath"`
	StoragePath                string          `json:"storagePath"`
	TempStoragePath			   string          `json:"tempStoragePath"`
	StoragePermission          string          `json:"storagePermission"`
	NotifyServerPath           string          `json:"notifyServer"`
	Db                         DBConfiguration `json:"db"`
	TempLinkValidationDuration int             `json:"linkDuration"`
	ServeDirectory             string          `json:"serveDirectory"`
	ServeUrl                   string          `json:"serveUrl"`
	OwnerGid                   int             `json:"ownerGid"`
	OwnerUid                   int             `json:"ownerUid"`
	RemoveInterval             int             `json:"removeInterval"`
	CacheSubDirName            string          `json:"cacheSubDirName"`
	CacheAttributeFileName     string          `json:"cacheAttributeFileName"`
	CacheCoverFileName         string          `json:"cacheCoverFileName"`
	CoverSize                  string          `json:"coverSize"`
	CoverQuality               string          `json:"coverQuality"`
	ReferenceStoragePath       string         `json:"referenceStorePath"`
	FileHashLen                int             `json:"fileHashLen"`

}

const PLC_FORMAT = ".plc"
const EPUB_FORMAT = ".epub"
const PDF_FORMAT = ".pdf"
const EPUB_COVER_IMAGE = "cover"
const MEGABYTE = 1.0 << (10 * 3)
