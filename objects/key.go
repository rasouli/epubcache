package objects

import (
	"github.com/jinzhu/gorm"
	"errors"
)

type FileKey struct {
	gorm.Model
	Etag string `gorm:"index:idx_filekey"`
	User string `gorm:"index:idx_filekey"`
	Key  string `gorm:"index:idx_filekey"`
	Hash string `gorm:"index:idx_filekey"`
}

func (*FileKey) TableName() string {
	return "file_keys"
}

type KeyManager struct {
	Db *gorm.DB
}

func (km *KeyManager) PrepareDatabase() {
	if !km.Db.HasTable(&FileKey{}) {
		km.Db.CreateTable(&FileKey{})
	}
}

func (km *KeyManager) PersistKey(key *FileKey) error {

	keys := []FileKey{}
	needle := FileKey{Etag: key.Etag, User:key.User, Hash: key.Hash}
	err := km.Db.Where(needle).Find(&keys).Error

	if err != nil{
		return err
	}

	if len(keys) > 1 {
		return errors.New("Multiple Keys found!")
	}

	// update the key in place
	onlyKey := key
	update := false
	if len(keys) == 1 {
		onlyKey = &keys[0]
		onlyKey.Key = key.Key
		update = true
	}

	tx := km.Db.Begin()
	if update {
		err = tx.Save(onlyKey).Error
	} else {
		err = tx.Create(onlyKey).Error
	}

	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (km *KeyManager) FindKey(username string , etag string) (*FileKey, error) {
	temp := &FileKey{}
	needle := &FileKey{User:username, Etag:etag}
	err := km.Db.Where(needle).First(temp).Error
	return temp, err
}

func (km *KeyManager) FindAllUserKeys(username string) ([]FileKey, error) {
	c := &FileKey{}
	c.User = username
	result := []FileKey{}
	err := km.Db.Where(c).Find(&result).Error
	return result, err
}

func (km *KeyManager) FindUserKeysByEtag(username string, etags []string) ([]FileKey, error) {
	result := []FileKey{}
	err := km.Db.Where("user = (?) AND etag in (?)", username ,etags).Find(&result).Error
	return result, err
}


func (km *KeyManager) DeleteFileKey(key *FileKey) error {

	err := km.Db.Delete(key).Error
	return err
}

func (km *KeyManager) Close() error {
	err := km.Db.Close()
	return err
}
