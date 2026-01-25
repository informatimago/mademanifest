#!/bin/bash

# Test script for mademake-engine unit tests

echo "Running unit tests for mademake-engine..."

# Navigate to project directory
cd src/phase-2/mademanifest-engine

echo "1. Running all unit tests..."
go test ./pkg/...

echo "2. Running integration tests..."
go test -run TestFullPipeline

echo "3. Running tests with verbose output..."
go test -v ./pkg/...

echo "4. Running tests with coverage..."
go test -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt

echo "5. Running specific package tests..."

# Test each package individually
echo "  - Astronomy package:"
go test ./pkg/astronomy

echo "  - Ephemeris package:"
go test ./pkg/ephemeris

echo "  - Astrology package:"
go test ./pkg/astrology

echo "  - Human Design package:"
go test ./pkg/human_design

echo "  - Gene Keys package:"
go test ./pkg/gene_keys

echo "All tests completed!"
