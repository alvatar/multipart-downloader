package multipartdownloader

import (
	"errors"
	"fmt"
	//"io/ioutil"
	"log"
	"net/http"
	"sync"
	"strconv"
	"time"
)

// Info gathered from different sources
type URLInfo struct {
	url string
	fileLength int64
	connSuccess bool
	statusCode int
}

// The file downloader
type MultiDownloader struct {
	urls []string            // A list of all sources for the file
	nConns uint              // The number of max concurrent connections to use
	fileLength int64         // The size of the file. It could be larger than 4GB.
	timeout time.Duration    // Timeout for all connections
}

func NewMultiDownloader(urls []string, nConns uint, timeout time.Duration) *MultiDownloader {
	return &MultiDownloader{urls: urls, nConns: nConns, timeout: timeout}
}

// Get the info of the file, using the HTTP HEAD request
func (dldr *MultiDownloader) GatherInfo() (err error) {
	if len(dldr.urls) == 0 {
		return errors.New("No URLs provided")
	}

	var wg sync.WaitGroup
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
		if err != nil {
			log.Println("Error reading Content-Length from HTTP header")
			flen = 0
		}
		results <- URLInfo{
			url: url,
			fileLength: flen,
			connSuccess: true,
			statusCode: resp.StatusCode,
		}
		wg.Done()
	}
	wg.Add(len(dldr.urls))
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

	// Wait for all processes
	// XXX This probably isn't necessary as they were blocked by the channel
	wg.Wait()

	// Check that all sources agree on file length
	commonFileLength := resArray[0].fileLength
	for _, r := range resArray[1:] {
		if r.fileLength != commonFileLength {
			return errors.New("URLs must point to the same file")
		}
	}
	dldr.fileLength = commonFileLength

	LogVerbose("File length: ", dldr.fileLength)

	return
}
