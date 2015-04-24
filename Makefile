.PHONY: clean

all:
	go build cmd/godl.go

test:
	go test

clean:
	rm godl
