package multipartdownloader

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Info gathered from different sources
type URLInfo struct {
	url string
	fileLength int64
	etag string
	connSuccess bool
	statusCode int
}

// The file downloader
type MultiDownloader struct {
	urls []string            // A list of all sources for the file
	nConns uint              // The number of max concurrent connections to use
	timeout time.Duration    // Timeout for all connections
	fileLength int64         // The size of the file. It could be larger than 4GB.
	fileName string          // The output filename
	etag string              // The etag (if available) of the file
}

func NewMultiDownloader(urls []string, nConns uint, timeout time.Duration) *MultiDownloader {
	return &MultiDownloader{urls: urls, nConns: nConns, timeout: timeout}
}

// Get the info of the file, using the HTTP HEAD request
func (dldr *MultiDownloader) GatherInfo() (err error) {
	if len(dldr.urls) == 0 {
		return errors.New("No URLs provided")
	}

	results := make(chan URLInfo)
	//defer close(results)

	// Connect to all sources concurrently
	getHead := func (url string) {
		client := http.Client{
			Timeout: time.Duration(dldr.timeout),
		}
		resp, err := client.Head(url)
		if err != nil {
			results <- URLInfo{url: url, connSuccess: false, statusCode: 0}
			return
		}
		defer resp.Body.Close()
		flen, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 0, 64)
		etag := resp.Header.Get("Etag")
		if err != nil {
			log.Println("Error reading Content-Length from HTTP header")
			flen = 0
		}
		results <- URLInfo{
			url: url,
			fileLength: flen,
			etag: etag,
			connSuccess: true,
			statusCode: resp.StatusCode,
		}
	}
	for _, url := range dldr.urls {
		go getHead(url)
	}

	// Gather the results and return if something is wrong
	resArray := make([]URLInfo, len(dldr.urls))
	for i := 0; i < len(dldr.urls); i++ {
		r := <-results
		resArray[i] = r
		if !r.connSuccess || r.statusCode != 200 {
			return errors.New(fmt.Sprintf("Failed connection to URL %s", resArray[i].url))
		}
	}

	// Check that all sources agree on file length and Etag
	// Empty Etags are also accepted
	commonFileLength := resArray[0].fileLength
	commonEtag := resArray[0].etag
	for _, r := range resArray[1:] {
		if r.fileLength != commonFileLength || (len(r.etag) != 0 && r.etag != commonEtag) {
			return errors.New("URLs must point to the same file")
		}
	}
	dldr.fileLength = commonFileLength
	dldr.etag = commonEtag
	dldr.fileName = URLtoFileName(resArray[0].url)

	LogVerbose("File length: ", dldr.fileLength)
	LogVerbose("Etag: ", dldr.etag)

	return
}

// Prepare the file used for writing the blocks of data
func (dldr *MultiDownloader) SetupFile(fileName string) (os.FileInfo, error) {
	var file *os.File
	var err error
	if fileName == "" {
		file, err = os.Create(dldr.fileName)
	} else {
		file, err = os.Create(fileName)
	}
	if err != nil {
		return nil, err
	}

	err = file.Truncate(dldr.fileLength)
	fileInfo, err := file.Stat()
	return fileInfo, err
}

func URLtoFileName(urlStr string) string {
	url, err := url.Parse(urlStr)
	if err != nil {
		return "downloaded-file"
	}
	log.Println(url.Path)
	return "fileTest"
}
