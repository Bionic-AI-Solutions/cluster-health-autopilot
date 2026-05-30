# syntax=docker/dockerfile:1.7
#
# Multi-stage build for the `cha` binary.
# - builder: Go 1.26 toolchain, full source tree, compiles a static binary.
# - runtime: distroless/static — no shell, no package manager, ~2 MB layer
#            on top of the binary. Smallest reasonable attack surface.

FROM golang:1.26-alpine AS builder
WORKDIR /src

# Pre-fetch modules so the build layer cache is stable across source-only edits.
COPY go.mod go.sum ./
RUN go mod download

# Now the source.
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY catalog ./catalog
COPY api ./api

# VERSION may be overridden at build time:
#   docker build --build-arg VERSION=v0.1.0 ...
ARG VERSION=dev
ARG COMMIT=unknown

# Static binary: CGO disabled, target linux/amd64 by default but BUILDPLATFORM
# is honored when building via buildx for multi-arch.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/cha \
    ./cmd/cha
# cha-operator — controller-runtime manager (v1.8 Phase 1b). Same
# image hosts both binaries so chart installs don't need a second
# imagePullSecret; the operator Deployment overrides the command.
RUN go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/cha-operator \
    ./cmd/cha-operator

# ---- runtime ----
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/cha /usr/local/bin/cha
COPY --from=builder /out/cha-operator /cha-operator
# Numeric UID matches distroless's `nonroot` user (65532). MUST stay
# numeric — kubelet's `runAsNonRoot: true` admission rejects images
# whose USER directive is a non-numeric name, since it can't verify
# the resolved UID is non-zero without running the image.
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cha"]
