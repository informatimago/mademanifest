# Compile the pinned IANA tzdata source into a runnable zoneinfo
# directory using the system zic.  The Docker builder stage does the
# same thing inside the image; this Makefile exists so developers can
# reproduce the canonical zoneinfo locally for tests.
#
# Usage:
#   make zoneinfo            # writes ./zoneinfo/  (and ./zoneinfo/+VERSION)
#   make verify              # checks SHA512SUMS + PGP signature
#   make verify-sha          # SHA-512 only
#   make verify-sig          # PGP signature only
#   make clean               # removes ./zoneinfo and the extracted source
#
# Requires: shasum (or sha512sum), tar, gpg, zic (Debian: tzdata
# package; macOS: bundled with the system).

TZDATA_TAR  := tzdata2026a.tar.gz
TZDATA_SIG  := $(TZDATA_TAR).asc
TZDATA_VER  := 2026a
ZONEINFO    := zoneinfo
SRCDIR      := src
ZIC         ?= zic
SHA512_TOOL ?= $(shell command -v shasum >/dev/null 2>&1 && echo "shasum -a 512 -c" || echo "sha512sum -c")
GPG         ?= gpg

# Pinned signing key + full fingerprint.  The key file
# (eggert.pgp.key) is committed alongside the tarball; the
# fingerprint pin is the authoritative anchor — verify-sig refuses
# to accept any other key, even one that imports under the same key
# ID, so a hostile upstream substitution would fail the build.
SIGNING_KEY := eggert.pgp.key
SIGNING_FPR := 7E3792A9D8ACF7D633BC1588ED97E90E62AA7E34

# Ephemeral keyring used during verification.  We never touch the
# user's default keyring so signature verification cannot be
# influenced by previously trusted (or compromised) keys on the
# build host.
GPG_KEYRING := .verify.gpg

# Standard set of IANA source files passed to zic.  Mirrors the
# upstream Makefile's PRIMARY_YDATA + YDATA + NDATA list, minus the
# "backzone" / "factory" extras the canon does not need.
ZIC_INPUTS := africa antarctica asia australasia europe \
              northamerica southamerica etcetera backward

.PHONY: zoneinfo verify verify-sha verify-sig clean

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

verify: verify-sha verify-sig

verify-sha:
	$(SHA512_TOOL) SHA512SUMS

verify-sig:
	@rm -f $(GPG_KEYRING)
	$(GPG) --no-default-keyring --keyring ./$(GPG_KEYRING) \
	       --import $(SIGNING_KEY)
	@imported_fpr=$$($(GPG) --no-default-keyring --keyring ./$(GPG_KEYRING) \
	                  --with-colons --fingerprint \
	                | awk -F: '/^fpr:/ {print $$10; exit}'); \
	if [ "$$imported_fpr" != "$(SIGNING_FPR)" ]; then \
	    echo "ERROR: imported key fingerprint $$imported_fpr != pinned $(SIGNING_FPR)"; \
	    rm -f $(GPG_KEYRING); \
	    exit 1; \
	fi
	$(GPG) --no-default-keyring --keyring ./$(GPG_KEYRING) \
	       --status-fd 1 --verify $(TZDATA_SIG) $(TZDATA_TAR) \
	       | grep -q "^\[GNUPG:\] VALIDSIG $(SIGNING_FPR) "
	@rm -f $(GPG_KEYRING)
	@echo "PGP signature OK (signed by $(SIGNING_FPR))"

clean:
	rm -rf $(SRCDIR) $(ZONEINFO) $(GPG_KEYRING)
