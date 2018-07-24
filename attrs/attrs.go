package attrs

import (
	"path/filepath"
	"os"
	"encoding/json"
	"bufio"
	"fmt"
	"encoding/base64"
	"io/ioutil"
	"os/exec"
	"epubcache/epub"
	"strings"
	"path"
	"errors"
	xpath "gopkg.in/xmlpath.v2"

)

func AttributeFileExists(storePath string, cacheSubDir string, attributeFileName string) bool {
	attributeFileAbsPath := filepath.Join(storePath, cacheSubDir, attributeFileName)
	_, err := os.Stat(attributeFileAbsPath)

	if os.IsNotExist(err) {
		return false
	}

	return true
}

func GetMediaAttributes(storePath string, cacheSubDir string, attributeFileName string) (map[string]string, error) {
	attributes := make(map[string]string)
	attributeFileAbsPath := filepath.Join(storePath, cacheSubDir, attributeFileName)

	file, err := os.OpenFile(attributeFileAbsPath, os.O_RDONLY, 0666)

	if err != nil {
		return attributes, err
	}

	defer file.Close()

	data, err := ioutil.ReadAll(file)

	if err != nil {
		return attributes, err
	}

	err = json.Unmarshal(data, &attributes)

	if err != nil {
		return attributes, err
	}
	return attributes, nil

}

func GetImageBase64(coverImagePath string) (string, error) {

	base64Str := ""

	fileInfo, err := os.Stat(coverImagePath)

	if os.IsNotExist(err) {
		return base64Str, nil // suppress the non existence error.
	}

	imageFormat := filepath.Ext(coverImagePath)[1:] // remove the dot

	file, err := os.OpenFile(coverImagePath, os.O_RDONLY, 0666)

	if err != nil {
		return base64Str, err
	}

	defer file.Close()

	fileSize := fileInfo.Size()
	buffer := make([]byte, fileSize)

	bufReader := bufio.NewReader(file)
	bufReader.Read(buffer)

	encoded := base64.StdEncoding.EncodeToString(buffer)
	base64Str = fmt.Sprintf("data:image/%s;base64,%s", imageFormat, encoded)

	return base64Str, err

}

func HasCoverImage(attributes map[string]string) bool {
	if val, ok := attributes["_cover"]; ok {

		if val == "" {
			return false
		} else {
			return true
		}
	}

	return false
}

func GetCoverImageAbsPath(storePath string, cacheSubDir string, attributes map[string]string) string {
	metaDir := filepath.Join(storePath, cacheSubDir)
	if val, ok := attributes["_cover"]; ok {

		return filepath.Join(metaDir, val)
	} else {
		return ""
	}
}

func RenderPDF(storePath string, pdfName string, metaSubDir string, coverName string, quality string, size string) error {

	pdfSrc := filepath.Join(storePath, pdfName)
	srcFileFirstPage := fmt.Sprintf("%s[0]", pdfSrc)
	destFile := filepath.Join(storePath, metaSubDir, coverName)

	return ChangeImageSize(srcFileFirstPage,destFile,quality, size)
}

func ChangeImageSize(source string, dest string, quality string, size string) error {
	cmd := "convert"
	cmdArgs := []string{source, "-quality", quality,"-background",  "white", "-alpha",  "remove", "-resize", size, dest}
	err := exec.Command(cmd, cmdArgs...).Run()

	return err
}

func GetEpubMetadataFromFile(book *epub.Book, path string) map[string]string {
	attributes := map[string]string{}

	// these attribute names are same as bookshop
	attributes["title"] = strings.Join(book.Opf.Metadata.Title, " ")
	attributes["description"] = strings.Join(book.Opf.Metadata.Description, " ")
	var authors []string
	for _, author := range book.Opf.Metadata.Creator {
		authors = append(authors, author.Data)
	}
	attributes["author"] = strings.Join(authors, ",")
	attributes["text"] = attributes["description"]
	attributes["subject"] = strings.Join(book.Opf.Metadata.Subject, ",")
	attributes["publisher"] = strings.Join(book.Opf.Metadata.Publisher, ",")

	var dates []string
	for _, date := range book.Opf.Metadata.Date {
		dates = append(dates, date.Data)
	}

	attributes["publishDate"] = strings.Join(dates, ",")

	return attributes
}

func GetCoverImagePathForEpub(book *epub.Book, extractPath string) string {

	rootFile := book.Container.Rootfile.Path
	pathUnderEpub := filepath.Dir(rootFile)

	if pathUnderEpub == "." {
		pathUnderEpub = ""
	}
	// in all manifests look
	for _, manifest := range book.Opf.Manifest {

		//if manifest.ID == objects.EPUB_COVER_IMAGE {
		if strings.Contains(manifest.MediaType, "image") {
			if strings.Contains(strings.ToLower(manifest.ID), "cover") || strings.Contains(strings.ToLower(manifest.Properties), "cover") {
				return filepath.Join(extractPath, pathUnderEpub, manifest.Href)
			}
		}
		//}

	}

	return ""
}

func GetCoverImageFromXHtml(book *epub.Book, extractPath string) (string, error) {

	opfFileRelativePath := ""
	for _, f := range book.Files() {

		if strings.HasSuffix(strings.ToLower(f),"opf") {
			opfFileRelativePath = f
			break
		}
	}


	if opfFileRelativePath == "" {
		return "" , errors.New("OPF file not found")
	}

	opfFileAbsPath := path.Join(extractPath,opfFileRelativePath)

	opfFile, err := os.Open(opfFileAbsPath)

	if err != nil {
		return "", err
	}

	defer opfFile.Close()

	node, err := xpath.Parse(opfFile)

	if err != nil {
		return "", err
	}

	xquery := xpath.MustCompile("//reference[@type=\"cover\"]/@href")
	refIter  := xquery.Iter(node)

	refAvailable := refIter.Next()
	if !refAvailable {
		return "", nil
	}

	coverRelativePath := refIter.Node().String()
	coverAbsPath := path.Join(path.Dir(opfFileAbsPath), coverRelativePath)

	coverReader, err := os.Open(coverAbsPath)
	if err != nil {
		return "" , err
	}

	defer coverReader.Close()

	coverNode , err := xpath.Parse(coverReader)

	if err != nil {
		return "", err
	}
	imageXQuery := xpath.MustCompile("//image/@href")
	imgXQuery := xpath.MustCompile("//img/@src")

	imageNodeIter := imageXQuery.Iter(coverNode)
    imgNodeIter := imgXQuery.Iter(coverNode)

	hasImageCover := imageNodeIter.Next()
	hasImgCover := imgNodeIter.Next()

	imageRelativePath := ""
	if hasImageCover {
		imageRelativePath = imageNodeIter.Node().String()
	} else if hasImgCover {
		imageRelativePath = imgNodeIter.Node().String()
	}

	if imageRelativePath == "" {
		return "" , nil
	}


	return path.Join(path.Dir(coverAbsPath), imageRelativePath) ,nil
}
