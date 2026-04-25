FROM golang:1.22-bookworm AS builder

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        patch \
        rsync \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY swisseph-makefile-install.patch ./
COPY ephemeris ./ephemeris

RUN cd ephemeris/swisseph \
    && rm -rf swisseph-2.10.03 \
    && tar zxf swisseph-2.10.03.tar.gz \
    && cd swisseph-2.10.03 \
    && patch -p1 < ../../../swisseph-makefile-install.patch \
    && make PREFIX=/usr/local libswe.so libswe.a \
    && install -d /usr/local/lib /usr/local/include \
    && install -m 644 libswe.so libswe.a /usr/local/lib/ \
    && install -m 644 sweph.h swephexp.h /usr/local/include/

RUN mkdir -p /usr/local/share/swisseph \
    && rsync -a ephemeris/data/REQUIRED_EPHEMERIS_FILES/ /usr/local/share/swisseph/

# Phase 12 retired the legacy src/canon/ JSON directory.  The
# trinity HTTP path consumes only the compiled-in canon constants
# (pkg/canon/constants.go), so the image no longer ships any canon
# JSON files.
COPY mademanifest-engine ./mademanifest-engine

WORKDIR /build/mademanifest-engine
RUN CGO_LDFLAGS="-lm" go build -o /out/mademanifest-http ./cmd/httpserver

FROM debian:bookworm-slim

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libgcc-s1 \
        libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

ENV PORT=8080
ENV SE_EPHE_PATH=/usr/local/share/swisseph

WORKDIR /app

COPY --from=builder /usr/local/lib/libswe.so /usr/local/lib/libswe.so
COPY --from=builder /usr/local/share/swisseph /usr/local/share/swisseph
COPY --from=builder /out/mademanifest-http /usr/local/bin/mademanifest-http

RUN ldconfig

EXPOSE 8080

CMD ["/usr/local/bin/mademanifest-http"]
