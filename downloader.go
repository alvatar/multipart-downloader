package multipartdownloader

// A type respresenting a file downloader.
type MultiDownloader struct {
	urls []string   // A list of all sources for the file
	nConns uint     // The number of max concurrent connections to use
}

func NewMultiDownloader(urls []string, nConns uint) (dldr *MultiDownloader) {
	dldr = &MultiDownloader{urls: urls, nConns: nConns}
	return
}
