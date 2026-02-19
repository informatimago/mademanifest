PREFIX=/usr/local
help::

.PHONY: swisseph-prepare swisseph-compile swisseph-install swisseph-install-data \
	help prepare

help::
	@printf 'make prepare  # compiles swisseph and install it.\n'
	@printf 'make compile  # compiles the go program.\n'
	@printf 'make run      # produces an output JSON file from the Golden Test input.\n'
	@printf 'make diff     # diffs the produced output against the Golden Test expected output with no differences.\n'
	@printf 'make all      # all of the above.\n'
	@printf 'make help     # prints this help.\n'

all: prepare compile run diff

prepare: swisseph-install swisseph-install-data
compile:
run:
diff:

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
