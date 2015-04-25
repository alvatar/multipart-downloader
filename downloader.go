package multipartdownloader

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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

// Perform the concurrent download
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
func (dldr *MultiDownloader) CheckSHA256(sha256 string) (ok bool, err error) {
	return
}

// Check ETag as MD5SUM
// Returns ok = true if either the check was correct or no ETag is available, with
// reason as error
// If an MD5SUM is provided, compare with it. Otherwise, try with the ETag
func (dldr *MultiDownloader) CheckMD5(md5sum string) (ok bool, err error) {
	if dldr.etag == "" && md5sum == "" {
		return true, errors.New("HTTP response doesn't contain an MD5 ETag")
	}
	// Use the provided md5sum if available
	var compareMD5SUM string
	if md5sum != "" {
		compareMD5SUM = md5sum
	} else {
		compareMD5SUM = dldr.etag
	}

	// Open the file and get the size
	file, err := os.Open(dldr.partFilename)
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, _ := file.Stat()
	if err != nil {
		return false, err
	}
	filesize := info.Size()

	// Compute the MD5SUM reading chunks
	blocks := uint64(math.Ceil(float64(filesize) / float64(fileReadChunk)))
	hash := md5.New()
	for i := uint64(0); i < blocks; i++ {
		blocksize := int(math.Min(fileReadChunk, float64(filesize-int64(i*fileReadChunk))))
		buf := make([] byte, blocksize)
		file.Read(buf)
		io.WriteString(hash, string(buf))
	}
	computedMD5SUMbytes := hash.Sum(nil)

	// Compare the MD5SUM
	computedMD5SUM := fmt.Sprintf("%x", computedMD5SUMbytes)
	if computedMD5SUM == compareMD5SUM {
		return true, nil
	} else {
		logVerbose("Computed MD5SUM does not match: ETag/provided=", compareMD5SUM, " computed=", computedMD5SUM)
		return false, errors.New("HTTP response doesn't contain an MD5 ETag")
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
