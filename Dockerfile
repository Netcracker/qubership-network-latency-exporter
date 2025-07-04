# hadolint global ignore=DL3018
# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.24.4-alpine3.21 AS builder
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
FROM alpine:3.22.0

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
