.PHONY: clean

all:
	go install
	go build cmd/godl.go

test: all
	go test
	go test ./cmd

clean:
	rm godl
