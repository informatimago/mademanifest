# Makefile for mademanifest-engine tests

.PHONY: test test-unit test-integration

all:
	cd mademanifest-engine ; go build -o proof-of-capability ./cmd


test: test-unit test-integration
	$(MAKE) -C tests tests

test-unit:
	go test ./pkg/...

test-integration:
	go test -run TestFullPipeline

# Run all tests with coverage
test-coverage:
	go test -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run specific package tests
test-astronomy:
	go test ./pkg/astronomy

test-ephemeris:
	go test ./pkg/ephemeris

test-astrology:
	go test ./pkg/astrology

test-human_design:
	go test ./pkg/human_design

test-gene_keys:
	go test ./pkg/gene_keys

# Clean test artifacts
clean:
	rm -f coverage.txt
