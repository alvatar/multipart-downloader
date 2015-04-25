.PHONY: clean

all:
	go install
	go build cmd/godl.go

test: all
	@set -e; \
	STATUS=0; \
	go test || STATUS=$$?; \
	go test ./cmd || STATUS=$$?; \
	exit $$STATUS; \

clean:
	rm godl
