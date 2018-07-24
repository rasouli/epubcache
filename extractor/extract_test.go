package extractor

import (
	"testing"
	"os"
	"path/filepath"
)

func TestExtractEpubContent(t *testing.T) {
	sampleEpubPath := "/tmp/eptest/Metamorphosis-jackson.epub"
	extractDirectory := "/tmp/eptest/extract"


	err := ExtractEpubContent(sampleEpubPath, extractDirectory)
	if err != nil {
		t.Fatalf("could not extract epub file: %s", err)

	}

}

func TestGetEpubExtractPath(t *testing.T) {
	base := os.TempDir()
	user := "test_user"
	etag := "aaabbbcccdddd111"
	expectedEpubPath := filepath.Join(base,user,etag)
	epubExtractionPath := GetStorePath(base,user,etag)

	if epubExtractionPath != expectedEpubPath {
		t.Fatalf("expected tho resulting extract path be : %s \n got: %s", expectedEpubPath, epubExtractionPath)
	}
}