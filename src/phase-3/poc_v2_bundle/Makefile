PREFIX       = /usr/local
SE_EPHE_PATH = $(PREFIX)/share/swisseph
ROOT        := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
SRC         := $(ROOT)mademanifest-engine
GOLDEN      := $(ROOT)golden
TEST_FLAGS   = -count=1
NAME         = proof-of-capability-2
# DIFF_RELAX   = -twb
DIFF_RELAX   =
IMAGE_NAME   = mademanifest-poc-v2
IMAGE_WORK_DIR = /workspace/mademanifest/src/phase-3/poc_v2_bundle

export SE_EPHE_PATH


help::

.PHONY: swisseph-prepare swisseph-compile swisseph-install swisseph-install-data \
	help prepare compile run diff all \
	test test-unit test-integration \
    test-coverage test-verbose test-astronomy \
	test-ephemeris test-astrology \
	test-human_design test-gene_keys \
	clean \
	docker-image \
	docker-update \
	docker-compile \
	docker-run

help::
	@printf 'make prepare  # compiles swisseph and install it.\n'
	@printf 'make compile  # compiles the go program.\n'
	@printf 'make run      # produces an output JSON file from the Golden Test input.\n'
	@printf 'make diff     # diffs the produced output against the Golden Test expected output with no differences.\n'
	@printf 'make all      # all of the above.\n'
	@printf 'make test     # run tests.\n'
	@printf 'make help     # prints this help.\n'

all: prepare compile run diff

prepare: swisseph-install swisseph-install-data
compile: $(SRC)/$(NAME)
run:	 compile
	cd $(SRC) ; LD_LIBRARY_PATH=$${LD_LIBRARY_PATH}:/usr/local/lib ./$(NAME) -cd ../canon ../golden/GOLDEN_TEST_CASE_V1.json out.json ; cat out.json
diff:    compile test-integration

swisseph-prepare:
	cd ephemeris/swisseph ; rm -rf swisseph-2.10.03 ; tar zxf swisseph-2.10.03.tar.gz
	cd ephemeris/swisseph/swisseph-2.10.03 ; patch -p1 < ../../../swisseph-makefile-install.patch

swisseph-compile: swisseph-prepare
	make PREFIX=$(PREFIX) -C ephemeris/swisseph/swisseph-2.10.03

swisseph-install: swisseph-compile
	make PREFIX=$(PREFIX) -C ephemeris/swisseph/swisseph-2.10.03 install

swisseph-install-data:
	sudo rsync -HSWacvxz --progress --force --delete --delete-after \
		ephemeris/data/REQUIRED_EPHEMERIS_FILES/ $(PREFIX)/share/swisseph/

$(SRC)/$(NAME):
	cd $(SRC) ; CGO_LDFLAGS="-lm" go build -o $(NAME) ./cmd
	install_name_tool -change libswe.dylib @rpath/libswe.dylib $(SRC)/$(NAME)
	install_name_tool -add_rpath /usr/local/lib                $(SRC)/$(NAME)

test: test-unit test-integration

test-unit:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/...

test-integration-unix: $(SRC)/$(NAME)
	cd $(SRC) ; LD_LIBRARY_PATH=$${LD_LIBRARY_PATH}:/usr/local/lib ./$(NAME) -cd ../canon ../golden/GOLDEN_TEST_CASE_V1.json out.json \
	&& tr -d '\015' < ../golden/GOLDEN_TEST_CASE_V1.json > expected.json \
	&& diff $(DIFF_RELAX) --width=160 -y out.json expected.json

test-integration: $(SRC)/$(NAME)
	cd $(SRC) ; LD_LIBRARY_PATH=$${LD_LIBRARY_PATH}:/usr/local/lib ./$(NAME) -dos -cd ../canon ../golden/GOLDEN_TEST_CASE_V1.json out.json \
	&& ( diff $(DIFF_RELAX) --width=160 -y out.json ../golden/GOLDEN_TEST_CASE_V1.json > diff ; status=$$? ; if [ $$status -ne 0 ] ; then cat diff ; fi ; )


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
	rm -f coverage.txt out.json $(SRC)/$(NAME)


docker-image:
	docker build -f env/docker/Dockerfile.txt -t "$(IMAGE_NAME)" .

docker-update:
	docker run --rm -t \
	  -v "$$(pwd)":$(IMAGE_WORK_DIR) \
	  -w $(IMAGE_WORK_DIR) \
	  "$(IMAGE_NAME)" \
	  bash -lc "git pull ; make clean compile"


docker-compile:
	docker run --rm -t \
	  -v "$$(pwd)":$(IMAGE_WORK_DIR) \
	  -w $(IMAGE_WORK_DIR) \
	  "$(IMAGE_NAME)" \
	  bash -lc "make clean compile"

docker-run:
	docker run --rm -t \
	  -v "$$(pwd)":$(IMAGE_WORK_DIR) \
	  -w $(IMAGE_WORK_DIR) \
	  "$(IMAGE_NAME)" \
	  bash -lc "make run diff"
