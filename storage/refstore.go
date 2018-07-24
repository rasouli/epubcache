package storage

import (
	"os"
	"crypto/sha1"
	"io"
	"encoding/hex"
	"path"
	"epubcache/objects"
)


type  RefStore struct {
	Config *objects.Config
}

type RefrenceData struct {
	User string
	Etag string

}

// compute the hash of the file
func (rs *RefStore) ComputeHash(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}

	defer f.Close()

	hash := sha1.New()

	_, err = io.Copy(hash, f)

	if err != nil {
		return "", err
	}

	hashBytes := hash.Sum(nil)[:rs.Config.FileHashLen]
	return hex.EncodeToString(hashBytes), err

}

// check if the given hash exists in directory
func (rs *RefStore) HashDirExists(hash string) bool {
	hashDir := path.Join(rs.Config.ReferenceStoragePath, hash)
	_, e := os.Stat(hashDir)

	if os.IsNotExist(e) {
		return false
	}

	return true
}



func (rs *RefStore) GetHashAbsDir(hash string) string {
	return path.Join(rs.Config.ReferenceStoragePath, hash)
}

//create hash dir of the file, create if it does not exists
func (rs *RefStore) GetHashDirOrCreate(hash string) (string, error) {



	if rs.HashDirExists(hash) {
		return hash, nil
	}

	err := os.MkdirAll(rs.GetHashAbsDir(hash), os.ModePerm)

	if err != nil {
		return "", err
	}

	return hash, nil
}
