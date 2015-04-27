package multipartdownloader

import (
	"bytes"
	"net"
	"net/http"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hydrogen18/stoppableListener"
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
	_, err := dldr.GatherInfo()
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
	_, err := dldr.GatherInfo()
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
		chunks []Chunk
	} {
		{125, 1, []Chunk{{0, 125},}},
		{125, 2, []Chunk{{0, 63}, {63, 125},}},
		{125, 3, []Chunk{{0, 42}, {42, 84}, {84, 125},}},
		{125, 4, []Chunk{{0, 32}, {32, 63}, {63, 94}, {94, 125},}},
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
	_, err := dldr.GatherInfo()
	failOnError(t, err)

	_, err = dldr.SetupFile("")
	failOnError(t, err)

	err = dldr.Download(nil)
	failOnError(t, err)
	if delete {
		defer func() {
			err = os.Remove(dldr.filename)
			failOnError(t, err)
		}()
	}

	// Load everything into memory and compare. Not efficient, but OK for testing
	f1, err := ioutil.ReadFile("test/quijote.txt")
	failOnError(t, err)
	f2, err := ioutil.ReadFile(dldr.filename)
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
		err := os.Remove(dldr.filename)
		failOnError(t, err)
	}()
	err := dldr.CheckSHA256("1e9bb1b16f8810e44d6d5ede7005258518fa976719bc2ed254308e73c357cfcc")
	if err != nil {
		t.Error(err)
	}
	err = dldr.CheckSHA256("wrong-hash")
	if err == nil {
		t.Error(err)
	}
}

// Test MD5SUM check
func TestCheckMD5SUMFile (t *testing.T) {
	dldr := downloadElQuijote(t, []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote.txt"}, 1, false)
	defer func() {
		err := os.Remove(dldr.filename)
		failOnError(t, err)
	}()
	// Compare manually with a MD5SUM generated with the command-line tool
	// Github's ETag doesn't reflect the MD5SUM
	err := dldr.CheckMD5("45bb5fc96bb4c67778d288fba98eee48")
	if err != nil {
		t.Error(err)
	}
	err = dldr.CheckMD5("wrong-hash")
	if err == nil {
		t.Error(err)
	}
}

// Test download with 1 remote source
func Test1SourceRemote (t *testing.T) {
	nConns := []int{1, 2, 5, 10}
	for _, n := range nConns {
		downloadElQuijote(t, []string{"https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote.txt"}, n, true)
	}
}

// Test download with 2 remote sources
func Test2SourcesRemote (t *testing.T) {
	nConns := []int{1, 2, 7, 19}
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

// Test download with a connection drop from one of the sources
// This test also triggers the server connection limit case
func TestConnectionDropLocal (t *testing.T) {
	shutdown := make(chan bool)

	go func() {
		originalListener, err := net.Listen("tcp", ":8081")
		if err != nil {
			panic(err)
		}

		sl, err := stoppableListener.New(originalListener)
		if err != nil {
			panic(err)
		}

		http.Handle("/", http.FileServer(http.Dir("./test")))
		server := http.Server{}

		go server.Serve(sl)

		// Stop this listener when the signal is received
		<- shutdown
		sl.Stop()
	}()

	go func() {
		originalListener, err := net.Listen("tcp", ":8082")
		if err != nil {
			panic(err)
		}

		sl, err := stoppableListener.New(originalListener)
		if err != nil {
			panic(err)
		}

		http.Handle("/quijote2", http.FileServer(http.Dir("./test")))
		server := http.Server{}

		go server.Serve(sl)

		// Stop this listener when the signal is received
		<- shutdown
		sl.Stop()
	}()

	// Wait for 50 milliseconds for listeners to be ready
	timer := time.NewTimer(time.Millisecond * 50)
	<- timer.C

	go downloadElQuijote(t, []string{
		"http://localhost:8081/quijote.txt",
		"http://localhost:8082/quijote2.txt",
	}, 2, true)

	// Wait to shutdown the listeners, hopefully in the middle of the transfer
	// TODO: Are transfers shut down off non-gracefully (as we wish)
	timer = time.NewTimer(time.Millisecond * 50)
	<- timer.C
	shutdown <- true
	shutdown <- true
}
