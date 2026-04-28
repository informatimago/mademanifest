# Phase 13 deployment hardening:
#   * Pinned base-image digests (reproducible builds; CVE auditability).
#   * Non-root runtime user `appuser` (UID 10001).
#   * HEALTHCHECK probes /healthz (matches the k8s liveness probe).
#   * SE_EPHE_PATH is intentionally a read-only data directory: the
#     engine never writes to it.  K8s pods set readOnlyRootFilesystem;
#     the only writable path needed at runtime is /tmp (tmpfs volume
#     in the manifest).
#
# Base image digests resolved via `docker pull --platform linux/amd64`
# at the time of this commit.  Re-resolve with the same command when
# bumping the tag; the digest pins MUST stay matched to the named tag.
FROM golang:1.22-bookworm@sha256:3d699e4d15d0f8f13c9195c0632a16702b8cbdece2955af1c23b37ae5d55a253 AS builder

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        patch \
        rsync \
        tzdata \
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

# Compile the canon-pinned IANA Time Zone Database (Document 03 / D20
# / A1 RESOLVED).  The source tarball + SHA-512 are tracked in
# src/tzdata/; we verify the checksum before extraction so a corrupted
# or substituted tarball aborts the build.  The compiled zoneinfo dir
# is consumed by the runtime stage via ZONEINFO; the +VERSION marker
# is read at boot by canon.AssertTZDBVersion.  We pass the standard
# IANA primary-data files (Olson set minus backzone/factory).
COPY tzdata ./tzdata
RUN cd tzdata \
    && sha512sum --check SHA512SUMS \
    && mkdir -p src \
    && tar -C src -xzf tzdata2026a.tar.gz \
    && test "$(cat src/version)" = "2026a" \
    && mkdir -p /usr/local/share/zoneinfo \
    && cd src \
    && zic -d /usr/local/share/zoneinfo \
        africa antarctica asia australasia europe \
        northamerica southamerica etcetera backward \
    && cp version /usr/local/share/zoneinfo/+VERSION

# Phase 12 retired the legacy src/canon/ JSON directory.  The
# trinity HTTP path consumes only the compiled-in canon constants
# (pkg/canon/constants.go), so the image no longer ships any canon
# JSON files.
COPY mademanifest-engine ./mademanifest-engine

WORKDIR /build/mademanifest-engine
RUN CGO_LDFLAGS="-lm" go build -o /out/mademanifest-http ./cmd/httpserver

FROM debian:bookworm-slim@sha256:f9c6a2fd2ddbc23e336b6257a5245e31f996953ef06cd13a59fa0a1df2d5c252

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        libgcc-s1 \
        libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

# Create the unprivileged runtime user used by the k8s securityContext
# (runAsUser: 10001).  We do not need a writable /home for the engine,
# so /home/appuser is created but stays root-owned and read-only to
# appuser; the only filesystem writes the engine performs at runtime
# are to a tmpfs /tmp mounted by the k8s manifest.
RUN groupadd --system --gid 10001 appuser \
    && useradd --system --uid 10001 --gid appuser --no-create-home --shell /sbin/nologin appuser

ENV PORT=8080
# SE_EPHE_PATH points at a read-only data directory installed by the
# builder stage above.  The engine reads .se1 files from this path
# but never writes to it; in k8s the path is on the read-only root
# filesystem.
ENV SE_EPHE_PATH=/usr/local/share/swisseph

# ZONEINFO points at the compiled IANA Time Zone Database installed
# by the builder stage.  The Go runtime consults $ZONEINFO before any
# system or embedded fallback, so time.LoadLocation reads the canon-
# pinned 2026a release in every container instance.  At startup,
# canon.AssertTZDBVersion reads <ZONEINFO>/+VERSION and aborts the
# process if it does not match canon.TZDBVersion.
ENV ZONEINFO=/usr/local/share/zoneinfo

WORKDIR /app

COPY --from=builder /usr/local/lib/libswe.so /usr/local/lib/libswe.so
COPY --from=builder /usr/local/share/swisseph /usr/local/share/swisseph
COPY --from=builder /usr/local/share/zoneinfo /usr/local/share/zoneinfo
COPY --from=builder /out/mademanifest-http /usr/local/bin/mademanifest-http

RUN ldconfig

USER 10001:10001

EXPOSE 8080

# HEALTHCHECK probes /healthz with curl.  The k8s liveness/readiness
# probes use the same path; this directive provides the same signal
# to the container runtime when the image is run outside k8s.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl --fail --silent --show-error http://localhost:8080/healthz || exit 1

CMD ["/usr/local/bin/mademanifest-http"]
