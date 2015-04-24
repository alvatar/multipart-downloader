package multipartdownloader

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func failOnError (t *testing.T, err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// NOTE: this test will fail if the file LICENSE is modified
func TestGatherInfo (t *testing.T) {
	// Gather remote sources info
	urls := []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/LICENSE"}
	dldr := NewMultiDownloader(urls, 1, time.Duration(5000) * time.Millisecond)
	err := dldr.GatherInfo()
	failOnError(t, err)

	// Get the local file info and test if they match
	file, err := os.Open("LICENSE") // For read access.
	failOnError(t, err)
	stat, err := file.Stat()
	failOnError(t, err)
	if stat.Size() != dldr.fileLength {
		t.Error("Remote and reference local file sizes do not match")
	}
}

func TestSetupfile (t *testing.T) {
	t.SkipNow()
	// Gather remote sources info
	urls := []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/LICENSE"}
	dldr := NewMultiDownloader(urls, 1, time.Duration(5000) * time.Millisecond)
	err := dldr.GatherInfo()
	failOnError(t, err)

	localFileInfo, err := dldr.SetupFile()
	failOnError(t, err)
	if localFileInfo.Size() != dldr.fileLength {
		t.Error("Downloaded and created local file sizes do not match")
	}
}

func Test1Source (t *testing.T) {
	t.SkipNow()
	nConns := []uint{1,2,5,10}
	for _, n := range nConns {
		t.Error(fmt.Sprintf("Failed downloading with a single source and %d connections", n))
	}
}

func Test2Sources (t *testing.T) {
	t.SkipNow()
	nConns := []uint{1, 2, 5, 30}
	for _, n := range nConns {
		t.Error(fmt.Sprintf("Failed downloading with 2 sources and %d connections", n))
	}
}

func Test3Sources (t *testing.T) {
	t.SkipNow()
	nConns := []uint{1, 2, 3, 5, 25, 26}
	for _, n := range nConns {
		t.Error(fmt.Sprintf("Failed downloading with 3 sources and %d connections", n))
	}
}
