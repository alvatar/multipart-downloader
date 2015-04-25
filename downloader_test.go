package multipartdownloader

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

func failOnError (t *testing.T, err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// MultiDownloader.GatherInfo() test
// NOTE: this test will fail if the file LICENSE diverges from the repository
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

// MultiDownloader.SetupFile() test
func TestSetupFile (t *testing.T) {
	// Gather remote sources info
	urls := []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/LICENSE"}
	dldr := NewMultiDownloader(urls, 1, time.Duration(5000) * time.Millisecond)
	err := dldr.GatherInfo()
	failOnError(t, err)

	// Create tmp file with custom name
	testFileName := "___testFile___"
	localFileInfo, err := dldr.SetupFile(testFileName)
	failOnError(t, err)
	// Remove the tmp file
	defer func() {
		err = os.Remove(testFileName)
		failOnError(t, err)
	}()
	if localFileInfo.Size() != dldr.fileLength {
		t.Error("Downloaded and created local file sizes do not match")
	}
}

func TestUrlToFilename (t *testing.T) {
	testTable := []struct {
		url string
		filename string
	} {
		{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/LICENSE",
			"LICENSE"},
		{"https://kernel.org/pub/linux/kernel/v4.x/linux-4.0.tar.xz",
			"linux-4.0.tar.xz"},
		{"https://kernel.org/pub/linux/kernel/v4.x/linux-4.0.tar.xz#frag-test",
			"linux-4.0.tar.xz"},
		{"https://kernel.org/pub/linux/kernel/v4.x/linux-4.0.tar.xz?type=animal&name=narwhal#nose",
			"linux-4.0.tar.xz"},
	}

	for _, test := range testTable {
		if urlToFilename(test.url) != test.filename {
			t.Fail()
		}
	}
}

func TestBuildChunks (t *testing.T) {
	testTable := []struct {
		fileLength int64
		nConns int
		chunks []chunk
	} {
		{125, 1, []chunk{{0, 125},}},
		{125, 2, []chunk{{0, 63}, {63, 125},}},
		{125, 3, []chunk{{0, 42}, {42, 84}, {84, 125},}},
		{125, 4, []chunk{{0, 32}, {32, 63}, {63, 94}, {94, 125},}},
	}
	for _, test := range testTable {
		urls := []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/LICENSE"}
		dldr := NewMultiDownloader(urls, test.nConns, time.Duration(1))
		dldr.fileLength = test.fileLength
		dldr.buildChunks()
		if !reflect.DeepEqual(dldr.chunks, test.chunks) {
			log.Println("Should be:", test.chunks)
			log.Println("Result is:", dldr.chunks)
			t.Fail()
		}
	}
}

// MultiDownloader.Download() tests
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

func TestCheckSHA256File (t *testing.T) {
	t.SkipNow()
}

func TestCheckETagFile (t *testing.T) {
	t.SkipNow()
}
