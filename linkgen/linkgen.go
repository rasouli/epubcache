package linkgen

import (
	obj "epubcache/objects"
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func New(config obj.DBConfiguration) (*obj.LinkGen, error) {
	gen := &obj.LinkGen{}
	dbLink := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database)

	db, err := gorm.Open("mysql", dbLink)

	gen.Db = db
	if err != nil {
		return gen, err
	}

	gen.PrepareDatabase()
	return gen, nil
}
