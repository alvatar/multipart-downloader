# Multi-part file downloader

Download a file with multiple connections and multiple sources simultaneously.

[![Build Status](https://travis-ci.org/alvatar/multipart-downloader.svg?branch=master)](https://travis-ci.org/alvatar/multipart-downloader)

## Installation

    make

This will build _godl_. The executable has no dependencies.

## Usage

    godl [flags ...] [urls ...]

    Flags:
        -n      Number of concurrent connections
        -S      File containing SHA-256 hash, or a SHA-256 string
        -E      Verify using Etag as MD5
        -v      Verbose output

