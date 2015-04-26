package multipartdownloader

import (
	"crypto/md5"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

const (
	tmpFileSuffix = ".part"
	fileWriteChunk = 1 << 12
	fileReadChunk = 1 << 12
)

// Info gathered from different sources
type URLInfo struct {
	url string
	fileLength int64
	etag string
	connSuccess bool
	statusCode int
}

// Chunk boundaries
type chunk struct {
	begin int64
	end int64
}

// The file downloader
type MultiDownloader struct {
	urls []string            // List of all sources for the file
	nConns int               // Number of max concurrent connections to use
	timeout time.Duration    // Timeout for all connections
	fileLength int64         // Size of the file. It could be larger than 4GB.
	filename string          // Output filename
	partFilename string      // Incomplete output filename
	etag string              // ETag (if available) of the file
	chunks []chunk           // A table of the chunks the file is divided into
}

func NewMultiDownloader(urls []string, nConns int, timeout time.Duration) *MultiDownloader {
	return &MultiDownloader{urls: urls, nConns: nConns, timeout: timeout}
}

// Get the info of the file, using the HTTP HEAD request
func (dldr *MultiDownloader) GatherInfo() (err error) {
	if len(dldr.urls) == 0 {
		return errors.New("No URLs provided")
	}

	results := make(chan URLInfo)
	defer close(results)

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
	dldr.etag = commonEtag[1:len(commonEtag)-1] // Remove the surrounding ""
	dldr.filename = urlToFilename(resArray[0].url)
	dldr.partFilename = dldr.filename + tmpFileSuffix

	logVerbose("File length: ", dldr.fileLength)
	logVerbose("File name: ", dldr.filename)
	logVerbose("Parts file name: ", dldr.partFilename)
	logVerbose("Etag: ", dldr.etag)

	return
}

// Prepare the file used for writing the blocks of data
func (dldr *MultiDownloader) SetupFile(filename string) (os.FileInfo, error) {
	if filename != "" {
		dldr.filename = filename
		dldr.partFilename = filename + tmpFileSuffix
	}

	file, err := os.Create(dldr.partFilename)
	if err != nil {
		return nil, err
	}

	// Force file size in order to write arbitrary chunks
	err = file.Truncate(dldr.fileLength)
	fileInfo, err := file.Stat()
	return fileInfo, err
}

// Internal: build the chunks table, deciding boundaries
func (dldr *MultiDownloader) buildChunks() {
	// The algorithm takes care of possible rounding errors splitting into chunks
	// by taking out the remainder and distributing it among the first chunks
	n := int64(dldr.nConns)
	remainder := dldr.fileLength % n
	exactNumerator := dldr.fileLength - remainder
	chunkSize := exactNumerator / n
	dldr.chunks = make([]chunk, n)
	boundary := int64(0)
	nextBoundary := chunkSize
	for i := int64(0); i < n; i++ {
		if remainder > 0 {
			remainder--
			nextBoundary++
		}
		dldr.chunks[i] = chunk{boundary, nextBoundary}
		boundary = nextBoundary
		nextBoundary = nextBoundary + chunkSize
	}
}

// Perform the multipart download
//
// This algorithm handles download splitting the file into n blocks. If a connection fails, it
// will try with other sources (as different sources may have different connection limits) then,
// if it still fails, it will wait until other process is done. Thus, nConns really means the
// MAXIMUM allowed connections, which will be tried at first and then adjusted.
// The alternative approach of dividing into nSize blocks and spawn threads requests from a pool
// of tasks has been discarded to avoid the overhead of performing potentially too many HTTP
// requests, as a result of each thread performing many requests instead of the minimum necessary.
//
// The designed algorithm tries to minimize the amount of successful HTTP requests.
//
// As a result of the approach taken, the number of concurrent connections can drop if no source
// is available to accomodate the request. In any case, setting a reasonable limit is left to the
// Take into consideration that some servers may ban your IP for some amount of time if you flood
// them with too many requests.
func (dldr *MultiDownloader) Download() (err error) {
	// Build the chunks table, necessary for constructing requests
	dldr.buildChunks()

	// Parallel download, wait for all to return
	var wg sync.WaitGroup
	downloadChunk := func(f *os.File, i int) {
		defer wg.Done()
		client := &http.Client{}
		// Select URL in a Round-Robin fashion
		selectedUrl := dldr.urls[i%len(dldr.urls)]

		// Send per-range requests
		req, err := http.NewRequest("GET", selectedUrl, nil)
		if err != nil {
			panic (err)
		}
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", dldr.chunks[i].begin, dldr.chunks[i].end))
		resp, err := client.Do(req)
		if err != nil {
			// TODO: should signal failure
			return
		}
		defer resp.Body.Close()

		// Read response and process it in chunks
		buf := make([]byte, fileWriteChunk)
		cursor := dldr.chunks[i].begin
		for {
			n, err := io.ReadFull(resp.Body, buf)
			if err == io.EOF {
				// TODO: should signal success
				return
			}
			// According to doc: "Clients of WriteAt can execute parallel WriteAt calls on the
			// same destination if the ranges do not overlap."
			_, errWr := f.WriteAt(buf[:n], cursor)
			if errWr != nil {
				// TODO: should signal this failure
				log.Fatal(errWr)
				return
			}
			cursor += int64(n)
		}
	}
	wg.Add(dldr.nConns)

	file, err := os.OpenFile(dldr.partFilename, os.O_WRONLY, 0666)
	if err != nil {
		return
	}

	for i := 0; i < dldr.nConns; i++ {
		go downloadChunk(file, i)
	}
	wg.Wait()
	return
}

// Check SHA-256 of downloaded file
func (dldr *MultiDownloader) CheckSHA256(sha256hash string) (ok bool, err error) {
	// Open the file and get the size
	file, err := os.Open(dldr.partFilename)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	// Compute the SHA256
	buf := make([] byte, fileReadChunk)
	hash := sha256.New()
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}

		if _, err := hash.Write(buf[:n]); err != nil {
			panic(err)
		}
	}
	computedSHA256bytes := hash.Sum(nil)

	// Compare the SHA256
	computedSHA256 := fmt.Sprintf("%x", computedSHA256bytes)

	if computedSHA256 == sha256hash {
		return true, nil
	} else {
		logVerbose("Computed SHA256 does not match: provided=", sha256hash, " computed=", computedSHA256)
		return false, nil
	}
}

// Check MD5SUM of downloaded file
func (dldr *MultiDownloader) CheckMD5(md5sum string) (ok bool, err error) {
	// Open the file and get the size
	file, err := os.Open(dldr.partFilename)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	// Compute the MD5SUM
	buf := make([] byte, fileReadChunk)
	hash := md5.New()
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}

		if _, err := hash.Write(buf[:n]); err != nil {
			panic(err)
		}
	}
	computedMD5SUMbytes := hash.Sum(nil)

	// Compare the MD5SUM
	computedMD5SUM := fmt.Sprintf("%x", computedMD5SUMbytes)

	if computedMD5SUM == md5sum {
		return true, nil
	} else {
		logVerbose("Computed MD5SUM does not match: provided=", md5sum, " computed=", computedMD5SUM)
		return false, nil
	}
}


////////////////////////////////////////////////////////////////////////////////
// Auxiliary functions

// Get the name of the file from the URL
func urlToFilename(urlStr string) string {
	url, err := url.Parse(urlStr)
	if err != nil {
		return "downloaded-file"
	}
	_, f := path.Split(url.Path)
	return f
}
