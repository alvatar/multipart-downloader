.PHONY: clean

all:
	go install
	go build cmd/godl.go

test: all
	go test

clean:
	rm godl
