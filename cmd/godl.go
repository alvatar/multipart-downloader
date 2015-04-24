package main

import (
	"flag"
	"log"
	"os"
	md "github.com/alvatar/multipart-downloader"
)


var (
	nConns   = flag.Uint("n", 1, "Number of concurrent connections")
	sha256   = flag.String("S", "", "File containing SHA-256 hash, or a SHA-256 string")
	useEtag  = flag.Bool("E", false, "Verify using Etag as MD5")
	verbose  = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()
	log.SetPrefix("godl: ")
	if len(flag.Args()) == 0 {
		log.Fatal("No urls provided")
		os.Exit(1)
	}
	if len(*sha256) == 0 {
		log.Println("No SHA-256 file or string provided")
	}
	log.Println(*nConns)
	log.Println(*sha256)
	log.Println(*useEtag)
	log.Println(*verbose)

	dldr := md.NewMultiDownloader(flag.Args(), *nConns)
	log.Println(dldr)
}
