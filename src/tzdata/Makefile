# Compile the pinned IANA tzdata source into a runnable zoneinfo
# directory using the system zic.  The Docker builder stage does the
# same thing inside the image; this Makefile exists so developers can
# reproduce the canonical zoneinfo locally for tests.
#
# Usage:
#   make zoneinfo            # writes ./zoneinfo/  (and ./zoneinfo/+VERSION)
#   make verify              # checks SHA512SUMS
#   make clean               # removes ./zoneinfo and the extracted source
#
# Requires: shasum (or sha512sum), tar, zic (Debian: tzdata package;
# macOS: bundled with the system).

TZDATA_TAR  := tzdata2026a.tar.gz
TZDATA_VER  := 2026a
ZONEINFO    := zoneinfo
SRCDIR      := src
ZIC         ?= zic
SHA512_TOOL ?= $(shell command -v shasum >/dev/null 2>&1 && echo "shasum -a 512 -c" || echo "sha512sum -c")

# Standard set of IANA source files passed to zic.  Mirrors the
# upstream Makefile's PRIMARY_YDATA + YDATA + NDATA list, minus the
# "backzone" / "factory" extras the canon does not need.
ZIC_INPUTS := africa antarctica asia australasia europe \
              northamerica southamerica etcetera backward

.PHONY: zoneinfo verify clean

zoneinfo: verify
	@rm -rf $(SRCDIR) $(ZONEINFO)
	@mkdir -p $(SRCDIR) $(ZONEINFO)
	tar -C $(SRCDIR) -xzf $(TZDATA_TAR)
	@if [ "$$(cat $(SRCDIR)/version)" != "$(TZDATA_VER)" ]; then \
	    echo "ERROR: tarball version $$(cat $(SRCDIR)/version) != expected $(TZDATA_VER)"; \
	    exit 1; \
	fi
	cd $(SRCDIR) && $(ZIC) -d $(CURDIR)/$(ZONEINFO) $(ZIC_INPUTS)
	cp $(SRCDIR)/version $(ZONEINFO)/+VERSION
	@echo "compiled $(ZONEINFO)/ from IANA $(TZDATA_VER)"

verify:
	$(SHA512_TOOL) SHA512SUMS

clean:
	rm -rf $(SRCDIR) $(ZONEINFO)
