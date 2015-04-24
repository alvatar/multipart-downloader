.PHONY: clean

all:
	go install
	go build cmd/godl.go

test: all
	go test || true
	go test ./cmd

clean:
	rm godl
