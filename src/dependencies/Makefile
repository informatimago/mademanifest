
# WGET= curl -C -O
WGET= wget -c # wget determines the offset itself.

SWISSEPH_GIT_URL = git@github.com:aloistr/swisseph.git
JPLFILES_GIT_URL = https://ssd.jpl.nasa.gov/ftp/eph/planets/Linux/
TZDB_URL = https://data.iana.org/time-zones/releases/tzdb-2025c.tar.lz


all:build-dependencies

.PHONY: get-dependencies build-dependencies swisseph-patched

build-dependencies: swisseph-patched
	make -C swisseph install

get-dependencies: swisseph-patched  # tzdb-2025c
# tzdb-2025c we don't use tzdb-2025c directly but some go library that integrates it.

swisseph-patched: swisseph
	cd swisseph ; if -r .swisseph-makefile-install.patched ; then true ; else patch -p1 < ../swisseph-makefile-install.patch ; touch .swisseph-makefile-install.patched ; fi

swisseph:
	git clone "$(SWISSEPH_GIT_URL)"
	cd swisseph/ephe/ ; $(WGET) "$(JPLFILES_GIT_URL)"'/de200/lnxm1600p2170.200'       && mv lnxm1600p2170.200      de200.eph
	cd swisseph/ephe/ ; $(WGET) "$(JPLFILES_GIT_URL)"'/de406/lnxm1600p2170.406'       && mv lnxm1600p2170.406      de406.eph
	cd swisseph/ephe/ ; $(WGET) "$(JPLFILES_GIT_URL)"'/de431/lnxm1600p2170.431'       && mv lnxm1600p2170.431      de431.eph
	cd swisseph/ephe/ ; $(WGET) "$(JPLFILES_GIT_URL)"'/de441/linux_m13000p17000.441'  && mv linux_m13000p17000.441 de441.eph

tzdb-2025c.tar.lz:
	$(WGET) $(TZDB_URL)

tzdb-2025c:tzdb-2025c.tar.lz
	tar Jxvf $^

