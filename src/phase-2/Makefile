# Makefile for mademanifest-engine tests

SRC = mademanifest-engine
TEST_FLAGS = -count=1

.PHONY: test test-unit test-integration

all:
	cd $(SRC) ; go build -o proof-of-capability ./cmd


test: test-unit test-integration

test-unit:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/...

test-integration: $(SRC)/proof-of-capability
	cd $(SRC) ; ./proof-of-capability ../data/input.json out.json && diff --width=160 -y out.json ../data/GOLDEN_TEST_CASE_V1.json


# Run all tests with coverage
test-coverage:
	cd $(SRC) ; go test $(TEST_FLAGS) -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt

# Run tests with verbose output
test-verbose:
	cd $(SRC) ; go test $(TEST_FLAGS) -v ./...

# Run specific package tests
test-astronomy:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/astronomy

test-ephemeris:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/ephemeris

test-astrology:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/astrology

test-human_design:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/human_design

test-gene_keys:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/gene_keys

# Clean test artifacts
clean:
	rm -f coverage.txt
