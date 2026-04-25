PREFIX       = /usr/local
SE_EPHE_PATH = $(PREFIX)/share/swisseph
ROOT        := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
SRC         := $(ROOT)mademanifest-engine
GOLDEN      := $(ROOT)golden
TEST_FLAGS   = -count=1
NAME         = mademanifest-engine
IMAGE_NAME   = mademanifest-engine-dev
IMAGE_WORK_DIR = /workspace/mademanifest/src

export SE_EPHE_PATH


help::

.PHONY: swisseph-prepare swisseph-compile swisseph-install swisseph-install-data \
	help prepare compile run diff all \
	test test-unit test-integration \
	test-integration-local test-integration-docker test-integration-k8s test-all \
	test-coverage test-verbose \
	doc user-manual-pdf \
	clean \
	docker-image \
	docker-update \
	docker-compile \
	docker-run

help::
	@printf 'make prepare                 # compile and install Swiss Ephemeris.\n'
	@printf 'make compile                 # compile cmd/httpserver into ./mademanifest-engine.\n'
	@printf 'make run                     # alias for test-integration-local (Trinity golden pack via local subprocess).\n'
	@printf 'make diff                    # alias for test-integration-local.\n'
	@printf 'make all                     # prepare + compile + test-integration-local.\n'
	@printf 'make test                    # unit tests (alias for test-unit).\n'
	@printf 'make test-unit               # go test ./pkg/...\n'
	@printf 'make test-integration-local  # Trinity harness: launch cmd/httpserver as subprocess.\n'
	@printf 'make test-integration-docker # Trinity harness: docker build + docker run.\n'
	@printf 'make test-integration-k8s    # Trinity harness: kind + kubectl (long-running).\n'
	@printf 'make test-all                # unit + local + docker + kubernetes.\n'
	@printf 'make doc                     # rebuild doc/user-manual.pdf from user-manual.md.\n'
	@printf 'make help                    # prints this help.\n'

# Phase 12 retired the file-based PoC CLI.  The "all" / "run" / "diff"
# targets are kept as aliases so existing muscle memory still works,
# but they now drive the canonical Trinity Golden Pack through the
# HTTP service rather than the legacy GOLDEN_TEST_CASE_V1.json
# file-based pipeline.

all: prepare compile test-integration-local

prepare: swisseph-install swisseph-install-data
compile: $(SRC)/$(NAME)

# `run` and `diff` are Phase 12 aliases for test-integration-local.
# Both used to drive the file-based PoC CLI against
# golden/GOLDEN_TEST_CASE_V1.json; that fixture and that CLI are gone.
run:  test-integration-local
diff: test-integration-local

# `test-integration` is a Phase 12 alias for test-integration-local.
# The original target shelled out to the file-based PoC CLI and
# diffed JSON against the now-deleted golden fixture.
test-integration: test-integration-local


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

# Phase 12 retargets the build at cmd/httpserver — the file-based
# entry point at cmd/main.go is gone.
$(SRC)/$(NAME):
	cd $(SRC) ; CGO_LDFLAGS="-lm" go build -o $(NAME) ./cmd/httpserver
	install_name_tool -change libswe.dylib @rpath/libswe.dylib $(SRC)/$(NAME)
	# pkg/sweph/rpath_cgo.go already embeds this rpath via cgo
	# LDFLAGS, so -add_rpath is idempotent-with-failure.  Swallow
	# the "would duplicate path" error so repeat builds still
	# succeed.
	install_name_tool -add_rpath /usr/local/lib $(SRC)/$(NAME) 2>/dev/null || true

test: test-unit

test-unit:
	cd $(SRC) ; go test $(TEST_FLAGS) ./pkg/...


# Trinity harness tests.  Each target builds its own runtime surface
# and drives it through the shared integration helpers.  The helper
# self-tests (no build tag) run as part of the ordinary `test-unit`
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


# Run all tests with coverage.
test-coverage:
	cd $(SRC) ; go test $(TEST_FLAGS) -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt

# Run tests with verbose output.
test-verbose:
	cd $(SRC) ; go test $(TEST_FLAGS) -v ./...

# Clean test artifacts.
clean:
	rm -f coverage.txt $(SRC)/$(NAME)


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
	  bash -lc "make run"
