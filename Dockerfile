# hadolint global ignore=DL3018
# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go sources
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY go.* /workspace/

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download -x

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o /build/network-latency-exporter ./cmd/

# Use alpine tiny images as a base
FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

# Set UID and user name
ENV USER_UID=2001 \
    USER_NAME=appuser \
    GROUP_NAME=appuser

COPY --from=builder --chown=${USER_UID} /build/network-latency-exporter /bin/network-latency-exporter

RUN apk add --no-cache --upgrade \
        mtr \
    && rm -rf /var/cache/apk/* \
    # Add user
    && addgroup ${GROUP_NAME} \
    && adduser -D -G ${GROUP_NAME} -u ${USER_UID} ${USER_NAME} \
    # Grant execute permissions for copied binary
    && chmod +x /bin/network-latency-exporter

USER ${USER_UID}

ENTRYPOINT [ "/bin/network-latency-exporter" ]
