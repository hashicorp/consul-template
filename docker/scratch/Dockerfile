### Builder
#
ARG GOVERSION=latest
FROM golang:${GOVERSION} AS builder
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ARG LD_FLAGS
ARG GOTAGS

WORKDIR "/go/src/github.com/hashicorp/consul-template"

COPY . .

RUN \
  CGO_ENABLED="0" \
  GO111MODULE=on \
  go build -mod vendor -a -o "/consul-template" \
    -ldflags "${LD_FLAGS}" -tags "${GOTAGS}"

### Final
#
FROM scratch
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ADD "https://curl.haxx.se/ca/cacert.pem" "/etc/ssl/certs/ca-certificates.crt"
COPY --from=builder "/consul-template" "/consul-template"
ENTRYPOINT ["/consul-template"]
