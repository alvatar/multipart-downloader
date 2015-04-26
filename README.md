# Multi-part file downloader

Download a file with multiple connections and multiple sources simultaneously.

[![Build Status](https://travis-ci.org/alvatar/multipart-downloader.svg?branch=master)](https://travis-ci.org/alvatar/multipart-downloader) [![Doc Status](https://godoc.org/github.com/alvatar/multipart-downloader?status.png)](https://godoc.org/github.com/alvatar/multipart-downloader)


## Installation

    # Build the executable
    make

    # Install as library
    go get github.com/alvatar/multipart-downloader

## Usage

    godl [flags ...] [urls ...]

    Flags:
        -n      Number of concurrent connections
        -S      A SHA-256 string to check the downloaded file
        -E      Verify using Etag as MD5
        -t      Timeout for all connections in milliseconds (default 5000)
        -o      Output file
        -v      Verbose output

## Usage as library

```go
urls := []string{
    "https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote.txt",
    "https://raw.githubusercontent.com/alvatar/multipart-downloader/master/test/quijote2.txt",}
nConns := 2
timeout := time.Duration(5000) * time.Millisecond
dldr := md.NewMultiDownloader(urls, nConns, timeout)

// Gather info from all sources
err := dldr.GatherInfo()

// Prepare the file to write downloaded blocks on it
_, err = dldr.SetupFile(*output)

// Perform download
err = dldr.Download()

err = dldr.CheckSHA256("1e9bb1b16f8810e44d6d5ede7005258518fa976719bc2ed254308e73c357cfcc")
err = dldr.CheckMD5("45bb5fc96bb4c67778d288fba98eee48")
```