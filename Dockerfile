# Build Phase 1a
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
FROM golang:1.18-alpine AS builder

ENV CGO_ENABLED 0

WORKDIR /tmp/builddir
COPY . /tmp/builddir

RUN go build -o /assets/check ./cmd/check \
    && go build -o /assets/in ./cmd/in \
    && go build -o /assets/out ./cmd/out


# Build Phase 1b: Generate latest ca-certificates
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
FROM debian:buster-slim AS certs

RUN \
  apt update && \
  apt install -y ca-certificates && \
  cat /etc/ssl/certs/* > /ca-certificates.crt


# Build Phase 2
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
FROM scratch

COPY --from=certs /ca-certificates.crt /opt/ssl/certs/
COPY --from=builder /assets /opt/resource

ENV HOME /root
ENV USER root
ENV PATH /opt/resource
ENV SSL_CERT_DIR=/opt/ssl/certs
