# Build Phase 1
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
FROM golang:1.18 AS builder

ENV CGO_ENABLED 0

RUN apt-get -qq update \
    && apt-get -yqq install upx

WORKDIR /tmp/builddir
COPY . /tmp/builddir

RUN go build -ldflags "-s -w -extldflags '-static'" -o /assets/check ./cmd/check \
    && strip /assets/check \
    && upx -q -9 /assets/check \
    && go build -ldflags "-s -w -extldflags '-static'" -o /assets/in ./cmd/in \
    && strip /assets/in \
    && upx -q -9 /assets/in \
    && go build -ldflags "-s -w -extldflags '-static'" -o /assets/out ./cmd/out \
    && strip /assets/out \
    && upx -q -9 /assets/out


# Build Phase 2
# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /assets /opt/resource

ENV HOME /root
ENV USER root
ENV PATH /opt/resource
ENV SSL_CERT_DIR=/opt/ssl/certs
