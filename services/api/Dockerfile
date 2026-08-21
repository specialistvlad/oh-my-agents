# syntax=docker/dockerfile:1

# ─── Stage 1: deps ────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS deps

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# ─── Stage 2: build ───────────────────────────────────────────────────────────
FROM deps AS build

COPY cmd/ cmd/
COPY internal/ internal/

# Populated by buildx from the target platform.
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /bin/server ./cmd/server

# ─── Stage 3: check ───────────────────────────────────────────────────────────
# Gates the release: the image only builds if `make check-ci` passes.
FROM deps AS check

ARG GOLANGCI_VERSION=v2.11.3

RUN apk add --no-cache make bash && \
    wget -q -O /tmp/golangci-lint-install.sh \
    https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh && \
    sh /tmp/golangci-lint-install.sh -b /usr/local/bin ${GOLANGCI_VERSION} && \
    go install gotest.tools/gotestsum@latest

COPY Makefile .golangci.yml ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY tests/ tests/

RUN make check-ci && touch /check-passed

# ─── Stage 4: release ─────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12 AS release

WORKDIR /app

COPY --from=check /check-passed /check-passed
COPY --from=build /bin/server /app/server

ARG COMMIT_HASH=unknown
ARG BUILD_TIME=unknown
ENV COMMIT_HASH=${COMMIT_HASH}
ENV BUILD_TIME=${BUILD_TIME}

EXPOSE 39170

ENTRYPOINT ["/app/server"]
