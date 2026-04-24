PREFIX       = /usr/local
SE_EPHE_PATH = $(PREFIX)/share/swisseph
ROOT        := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
SRC         := $(ROOT)mademanifest-engine
GOLDEN      := $(ROOT)golden
TEST_FLAGS   = -count=1
NAME         = mademanifest-engine
# DIFF_RELAX   = -twb
DIFF_RELAX   =
IMAGE_NAME   = mademanifest-poc-v2
IMAGE_WORK_DIR = /workspace/mademanifest/src

export SE_EPHE_PATH


help::

.PHONY: swisseph-prepare swisseph-compile swisseph-install swisseph-install-data \
	help prepare compile run diff all \
	test test-unit test-integration \
	test-integration-local test-integration-docker test-integration-k8s test-all \
    test-coverage test-verbose test-astronomy \
	test-ephemeris test-astrology \
	test-human_design test-gene_keys \
	doc user-manual-pdf \
	clean \
	docker-image \
	docker-update \
	docker-compile \
	docker-run

help::
	@printf 'make prepare                 # compiles swisseph and install it.\n'
	@printf 'make compile                 # compiles the go program.\n'
	@printf 'make run                     # produces an output JSON file from the Golden Test input.\n'
	@printf 'make diff                    # diffs the produced output against the Golden Test expected output with no differences.\n'
	@printf 'make all                     # all of the above.\n'
	@printf 'make test                    # run unit + PoC integration tests.\n'
	@printf 'make test-integration-local  # Trinity harness: launch cmd/httpserver as subprocess.\n'
	@printf 'make test-integration-docker # Trinity harness: docker build + docker run.\n'
	@printf 'make test-integration-k8s    # Trinity harness: kind + kubectl (long-running).\n'
	@printf 'make test-all                # unit + local + docker + kubernetes.\n'
	@printf 'make doc                     # rebuild doc/user-manual.pdf from user-manual.md.\n'
	@printf 'make help                    # prints this help.\n'

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
	# pkg/sweph/rpath_cgo.go already embeds this rpath via cgo
	# LDFLAGS, so -add_rpath is idempotent-with-failure.  Swallow
	# the "would duplicate path" error so repeat builds still
	# succeed.
	install_name_tool -add_rpath /usr/local/lib $(SRC)/$(NAME) 2>/dev/null || true

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


# Trinity harness tests (Phase 0+).  Each target builds its own runtime
# surface and drives it through the shared integration helpers.  The
# helper self-tests (no build tag) run as part of the ordinary `test`
# target above; the harness targets below opt into the runtime launch
# via build tags.

test-integration-local:
	cd $(SRC) ; LD_LIBRARY_PATH=$${LD_LIBRARY_PATH}:/usr/local/lib \
		go test $(TEST_FLAGS) -tags integration_local -timeout 120s ./integration/...

test-integration-docker:
	cd $(SRC) ; go test $(TEST_FLAGS) -tags integration_docker -timeout 600s ./integration/...

test-integration-k8s:
	cd $(SRC) ; go test $(TEST_FLAGS) -tags integration_k8s -timeout 900s ./integration/...

test-all: test-unit test-integration-local test-integration-docker test-integration-k8s


# Documentation build.
#
# user-manual.pdf is produced from user-manual.md with pandoc.  We use
# xelatex because it handles the Unicode in the code block output
# (arrows, fancy quotes) without needing a preamble dance.

DOC_DIR = $(ROOT)doc

doc: user-manual-pdf

user-manual-pdf: $(DOC_DIR)/user-manual.pdf

$(DOC_DIR)/user-manual.pdf: $(DOC_DIR)/user-manual.md
	pandoc \
	    --from=gfm+yaml_metadata_block \
	    --to=pdf \
	    --pdf-engine=xelatex \
	    -V geometry:margin=1in \
	    -V mainfont="Helvetica" \
	    -V monofont="Menlo" \
	    -V colorlinks=true \
	    --toc --toc-depth=2 \
	    -o $@ $<


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
