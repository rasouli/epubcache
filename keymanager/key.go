package keymanager

import (
	obj "epubcache/objects"
	"fmt"

	"github.com/jinzhu/gorm"
)

func New(config obj.DBConfiguration) (*obj.KeyManager, error) {
	manager := &obj.KeyManager{}
	dbLink := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database)


	db, err := gorm.Open("mysql", dbLink)

	manager.Db = db
	if err != nil {
		return manager, err
	}

	manager.PrepareDatabase()
	return manager, nil
}
