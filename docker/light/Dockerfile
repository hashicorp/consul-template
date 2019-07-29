### Builder
#
FROM golang:1.12 AS builder
LABEL maintainer "John Eikenberry <jae@zhar.net>"

ARG LD_FLAGS
ARG GOTAGS

WORKDIR "/go/src/github.com/hashicorp/consul-template"

COPY . .

RUN \
  CGO_ENABLED="0" \
  go build -a -o "/consul-template" -ldflags "${LD_FLAGS}" -tags "${GOTAGS}"

### Final
#
FROM alpine:latest
LABEL maintainer "John Eikenberry <jae@zhar.net>"

RUN apk add --no-cache ca-certificates
COPY --from=builder "/consul-template" "/bin/consul-template"
ENTRYPOINT ["/bin/consul-template"]
