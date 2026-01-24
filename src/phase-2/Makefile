all:
	cd mademanifest-engine ; go build -o proof-of-capability ./cmd

tests: all
	$(MAKE) -C tests tests

.PHONY: tests
