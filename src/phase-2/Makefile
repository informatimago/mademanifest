# Makefile for mademanifest-engine tests

SRC = mademanifest-engine

.PHONY: test test-unit test-integration

all:
	cd $(SRC) ; go build -o proof-of-capability ./cmd


test: test-unit test-integration
	$(MAKE) -C tests tests

test-unit:
	cd $(SRC) ; go test ./pkg/...

test-integration:
	go test -run TestFullPipeline

# Run all tests with coverage
test-coverage:
	cd $(SRC) ; go test -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt

# Run tests with verbose output
test-verbose:
	cd $(SRC) ; go test -v ./...

# Run specific package tests
test-astronomy:
	cd $(SRC) ; go test ./pkg/astronomy

test-ephemeris:
	cd $(SRC) ; go test ./pkg/ephemeris

test-astrology:
	cd $(SRC) ; go test ./pkg/astrology

test-human_design:
	cd $(SRC) ; go test ./pkg/human_design

test-gene_keys:
	cd $(SRC) ; go test ./pkg/gene_keys

# Clean test artifacts
clean:
	rm -f coverage.txt
