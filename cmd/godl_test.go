package main

import (
	"os/exec"
	"testing"
)

func TestNoArgs (t *testing.T) {
	cmd := exec.Command("../godl")
	err := cmd.Run()
	if err == nil { // exit code 0
		t.Error("Running godl without arguments should exit with error")
	}
}

func TestNoUrls (t *testing.T) {
	cmd := exec.Command("../godl", "-n 10")
	err := cmd.Run()
	if err == nil { // exit code 0
		t.Error("Running godl without an URL should exit with error")
	}
}

func TestWrongUrl (t *testing.T) {
	cmd := exec.Command("../godl", "http://example.com/nothing")
	err := cmd.Run()
	if err == nil { // exit code 0
		t.Error("Running godl with a wrong URL should exit with error")
	}
}

func TestUrl (t *testing.T) {
	cmd := exec.Command("../godl", "https://raw.githubusercontent.com/alvatar/multipart-downloader/master/Makefile")
	err := cmd.Run()
	if err != nil {
		t.Error("Running godl with an URL should be successful")
	}
}
