package multipartdownloader

import (
	"bytes"
	"io/ioutil"
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
		err = os.Remove(dldr.partFilename)
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

func downloadElQuijote(t *testing.T, urls []string, n int, delete bool) *MultiDownloader {
	// Gather remote sources info
	dldr := NewMultiDownloader(urls, n, time.Duration(5000) * time.Millisecond)
	err := dldr.GatherInfo()
	failOnError(t, err)

	_, err = dldr.SetupFile("")
	failOnError(t, err)

	err = dldr.Download()
	failOnError(t, err)
	if delete {
		defer func() {
			err = os.Remove(dldr.partFilename)
			failOnError(t, err)
		}()
	}

	// Load everything into memory and compare. Not efficient, but OK for testing
	f1, err := ioutil.ReadFile("test/quijote.txt")
	failOnError(t, err)
	f2, err := ioutil.ReadFile(dldr.partFilename)
	failOnError(t, err)

	if !bytes.Equal(f1, f2) {
		t.Fail()
	}
	return dldr
}

// Test SHA256 check
func TestCheckSHA256File (t *testing.T) {
	dldr := downloadElQuijote(t, []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote.txt"}, 1, false)
	defer func() {
		err := os.Remove(dldr.partFilename)
		failOnError(t, err)
	}()
	ok, _ := dldr.CheckSHA256("1e9bb1b16f8810e44d6d5ede7005258518fa976719bc2ed254308e73c357cfcc")
	if !ok {
		t.Fail()
	}
}

// Test MD5SUM check
func TestCheckMD5SUMFile (t *testing.T) {
	dldr := downloadElQuijote(t, []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote.txt"}, 1, false)
	defer func() {
		err := os.Remove(dldr.partFilename)
		failOnError(t, err)
	}()
	// Compare manually with a MD5SUM generated with the command-line tool
	// Github's ETag doesn't reflect the MD5SUM
	ok, _ := dldr.CheckMD5("45bb5fc96bb4c67778d288fba98eee48")
	if !ok {
		t.Fail()
	}
}

// MultiDownloader.Download() tests
func Test1Source (t *testing.T) {
	nConns := []int{1, 2, 5, 10}
	for _, n := range nConns {
		downloadElQuijote(t, []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote2.txt"}, n, true)
	}
}

func Test2Sources (t *testing.T) {
	nConns := []int{1, 2, 7}
	for _, n := range nConns {
		downloadElQuijote(t,
			[]string{
				"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote2.txt",
				"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote.txt",
			},
			n,
			true)
	}
}
