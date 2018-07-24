package extractor

import "os"
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/mholt/archiver"
	"epubcache/attrs"
)

// based on the given info, this function returns the path in which
// the epub must be extracted
func GetStorePath(base string, user string, etag string) string {

	extractionDir := filepath.Join(base, user, etag)
	return extractionDir
}

func GetUserStorePath(base string, user string) string {

	extractionDir := filepath.Join(base, user)
	return extractionDir
}

func GetMetaDataDir(base string, user string, etag string, metaSubDir string) string {
	metaDir := filepath.Join(base, user, etag, metaSubDir)
	return metaDir
}

// extracts given epub file to the destination directory
// the distenation directory must exist before hand.
func ExtractEpubContent(epubFile string, destDir string, ownerUser int, ownerGroup int) error {
	exists, err := FileExists(epubFile)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New(fmt.Sprintf("the epub file %s does not exist!", epubFile))
	}

	exists, err = FileExists(destDir)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New(fmt.Sprintf("destination directory %s does not exists", destDir))
	}

	err = archiver.Zip.Open(epubFile, destDir)

	if err != nil {
		return err
	}

	err = ChownR(destDir, ownerUser, ownerGroup)

	if err != nil {
		return err
	}

	return nil
}

// check whether a file or directory exists
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err != nil {
		return false, err
	}

	if os.IsNotExist(err) {
		return false, err
	}

	return true, nil
}

func ChownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}

func SaveMetaFile(metaDir string, metaFileName string, attributes map[string]string) error {
	metaFile := filepath.Join(metaDir, metaFileName)
	file, err := os.OpenFile(metaFile, os.O_CREATE|os.O_WRONLY, 0660)

	if err != nil {
		return err
	}

	defer file.Close()

	json.NewEncoder(file).Encode(&attributes)

	return nil
}

func CopyAndResizeCoverImage(metaDir string, srcCoverFile string, coverSaveName string, quality string, size string) (string, error) {

	coverFileFormat := filepath.Ext(srcCoverFile)
	coverFinalFileName := fmt.Sprintf("%s%s", coverSaveName, coverFileFormat)

	//srcFile, err := os.OpenFile(coverSrc, os.O_RDONLY, 0666)
	//if err != nil {
	//	return coverFinalFileName, err
	//}
	//
	//defer srcFile.Close()

	destCoverFile := filepath.Join(metaDir, coverFinalFileName)

	//destFile, err := os.OpenFile(destCoverFile, os.O_CREATE|os.O_WRONLY, 0666)

	//if err != nil {
	//	return coverFinalFileName, err
	//}

	//defer destFile.Close()

	//_, err = io.Copy(destFile, srcFile)

	// also copy the file at the convert
	err := attrs.ChangeImageSize(srcCoverFile, destCoverFile, quality, size)

	if err != nil {
		return coverFinalFileName, err
	}

	//if err != nil {
	//	return coverFinalFileName, err
	//}

	return coverFinalFileName, nil
}

func DeleteFile(path string) error {
	var err = os.Remove(path)
	return err
}

func CopyFile(source string, destination string) error {
	srcFile, err := os.OpenFile(source, os.O_RDONLY, 0666)

	if err != nil {
		return err

	}

	defer srcFile.Close()

	dstFile, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		return err
	}

	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)

	if err != nil {
		return err
	}

	return nil

}

func GetFileSizeInMB(filePath string) (float64, error) {

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	var fileSize float64 = 0.0
	if err != nil {
		return fileSize, err
	}

	fileInfo, err := file.Stat()

	if err != nil {
		return fileSize, err
	}

	bytes := fileInfo.Size()
	kb := (float64)(bytes / 1024)

	fileSize = (float64)(kb / 1024) // MB

	return fileSize, nil

}
