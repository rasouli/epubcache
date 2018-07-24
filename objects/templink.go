package objects

import (
	"os"
	"path/filepath"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
)

type TempLink struct {
	gorm.Model
	User   string    `gorm:"index:idx_link"`
	Expiry time.Time `gorm:"index:idx_expiry"`
	Etag   string    `gorm:"index:idx_link"`
	Link   string    `gorm:"index:idx_link"`
	Hash   string     `gorm:"index:idx_link"`
}

type LinkGen struct {
	Db *gorm.DB
}

func (lg *LinkGen) Remove(link *TempLink) error {
	tx := lg.Db.Begin()
	err := tx.Delete(link).Error

	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil

}

func (*TempLink) TableName() string {
	return "temp_links"
}

func (lg *LinkGen) PrepareDatabase() {
	if !lg.Db.HasTable(&TempLink{}) {
		lg.Db.CreateTable(&TempLink{})
	}
}

func (*LinkGen) CreateTempLink(user string, etag string, duration time.Duration, hash string) *TempLink {
	expiery := time.Now().Add(duration)
	uuid, _ := uuid.NewV4()
	link := uuid.String()

	return &TempLink{
		User:   user,
		Etag:   etag,
		Expiry: expiery,
		Link:   link,
		Hash:   hash,
	}
}

func (lg *LinkGen) PersistTempLink(link *TempLink, sourcePath string, servePath string) error {
	tx := lg.Db.Begin()
	err := tx.Create(link).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	symPath := filepath.Join(servePath, link.Link)

	err = os.Symlink(sourcePath, symPath)
	if err != nil {
		return err
	}
	return nil
}

func (lg *LinkGen) FetchTempLink(link *TempLink) (*TempLink, error) {
	temp := &TempLink{}
	err := lg.Db.Where(link).Where("expiry >= ?", time.Now()).First(temp).Error
	return temp, err
}

func (lg *LinkGen) GetExpierdLinks() ([]TempLink, error) {
	temp := []TempLink{}
	err := lg.Db.Where("expiry < ?", time.Now()).Find(&temp).Error
	return temp, err
}

func (lg *LinkGen) Close() error {
	err := lg.Db.Close()
	return err
}
