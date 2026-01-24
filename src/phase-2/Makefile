all:
	cd mademanifest-engine ; go build -o proof-of-capability ./cmd


test tests: all
	$(MAKE) -C tests tests

.PHONY: test tests
