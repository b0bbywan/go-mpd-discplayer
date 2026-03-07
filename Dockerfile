# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.0

# ── Builder base images ────────────────────────────────────────────────────────
# amd64: Trixie (native build)
# arm64/arm: RaspiOS (Trixie-based, runs natively via QEMU — no cross-compiler needed for CGO)
FROM debian:trixie-slim       AS base-amd64
FROM vascoguita/raspios:arm64 AS base-arm64
FROM vascoguita/raspios:armhf AS base-arm

# ── Builder ───────────────────────────────────────────────────────────────────
ARG TARGETARCH
FROM base-${TARGETARCH} AS builder

ARG GO_VERSION
ARG VERSION=dev
ARG TARGETARCH
ARG TARGETVARIANT

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    ca-certificates \
    gcc \
    libc6-dev \
    libdiscid-dev \
    libgudev-1.0-dev \
    libasound2-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Install Go for the running architecture.
# For ARM targets, Docker runs the raspios image natively via QEMU binfmt_misc —
# uname -m returns the actual target arch, so no cross-compiler is needed.
RUN set -eux; \
    GOARCH=$(case "$(uname -m)" in \
        x86_64)  echo "amd64"  ;; \
        aarch64) echo "arm64"  ;; \
        arm*)    echo "armv6l" ;; \
        *)       uname -m ;; \
    esac); \
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -O /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

ENV PATH="/usr/local/go/bin:$PATH"
ENV GOPATH="/go"
ENV CGO_ENABLED=1

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# GOARM is derived from TARGETVARIANT (v6 → 6, v7 → 7) for 32-bit ARM targets
RUN set -eux; \
    if [ "${TARGETARCH}" = "arm" ] && [ -n "${TARGETVARIANT}" ]; then \
        export GOARM="${TARGETVARIANT#v}"; \
    fi; \
    go build \
        -ldflags "-s -w -X 'github.com/b0bbywan/go-mpd-discplayer/cmd.AppVersion=${VERSION}'" \
        -o mpd-discplayer \
        .

# ── Binary export: extracted via --output type=local ─────────────────────────
FROM scratch AS export
COPY --from=builder /build/mpd-discplayer /mpd-discplayer

# ── Deb builder (extends binary builder) ─────────────────────────────────────
FROM builder AS deb-builder

ARG TARGETARCH
ARG TARGETVARIANT

RUN apt-get update && apt-get install -y --no-install-recommends \
    debhelper \
    dh-golang \
    && rm -rf /var/lib/apt/lists/*

# --no-check-builddeps: our custom Go satisfies golang-go but isn't an apt package
RUN set -eux; \
    if [ "${TARGETARCH}" = "arm" ] && [ -n "${TARGETVARIANT}" ]; then \
        export GOARM="${TARGETVARIANT#v}"; \
    fi; \
    dpkg-buildpackage -b -us -uc --no-check-builddeps; \
    mkdir -p /deb-output; \
    mv /mpd-discplayer_*.deb /deb-output/

# ── Deb export: extracted via --output type=local ────────────────────────────
FROM scratch AS deb-export
COPY --from=deb-builder /deb-output/ /
