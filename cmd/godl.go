package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	md "github.com/alvatar/multipart-downloader"
)

var (
	nConns   = flag.Uint("n", 1, "Number of concurrent connections")
	sha256   = flag.String("S", "", "File containing SHA-256 hash, or a SHA-256 string")
	useEtag  = flag.Bool("E", false, "Verify using ETag as MD5")
	timeout  = flag.Uint("t", 5000, "Timeout for all connections in milliseconds")
	output   = flag.String("o", "", "Output file")
	verbose  = flag.Bool("v", false, "Verbose output")
)

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()
	log.SetPrefix("godl: ")
	if len(flag.Args()) == 0 {
		log.Fatal("No URLs provided")
		os.Exit(1)
	}

	// Register signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		log.Fatal("Exit with incomplete download")
		os.Exit(1)
	}()

	if *verbose {
		log.Println("Initializing download with", *nConns, "concurrent connections")
	}

	// Initialize download
	dldr := md.NewMultiDownloader(flag.Args(), int(*nConns), time.Duration(*timeout) * time.Millisecond)
	md.SetVerbose(*verbose)

	// Gather info from all sources
	chunks, err := dldr.GatherInfo()
	exitOnError(err)

	// Prepare the file to write individual blocks on
	_, err = dldr.SetupFile(*output)
	exitOnError(err)

	// Perform download
	if *verbose {
		// Setup bar visualization
		v := NewProgress(chunks)
		err = dldr.Download(func(feedback []md.ConnectionProgress) {
			v.Update(feedback)
		})
	} else {
		err = dldr.Download(nil)
	}
	exitOnError(err)

	// Perform SHA256 check if requested
	if *sha256 != "" {
		err := dldr.CheckSHA256(*sha256)
		exitOnError(err)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		} else if *verbose {
			log.Println("SHA-256 checked successfully")
		}
	}

	// Perform MD5SUM from ETag if requested
	if *useEtag {
		err := dldr.CheckMD5(dldr.ETag)
		exitOnError(err)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		} else if *verbose {
			log.Println("MD5SUM checked successfully")
		}
	}
}
