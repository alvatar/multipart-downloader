package multipartdownloader

import (
	"log"
)

var verbose = false

// Verbose logging utility
func logVerbose(e ...interface{}) {
	if (verbose) {
		log.Print(e...)
	}
}

// Set verbosity for all log actions
func SetVerbose(verb bool) {
	verbose = verb
}

