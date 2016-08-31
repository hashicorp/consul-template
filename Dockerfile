FROM golang:1.6-alpine

RUN apk add --update make git bash

ARG GOOS=linux
ARG GOARCH=amd64

ENV GOOS=$GOOS \
    GOARCH=$GOARCH

RUN mkdir -p /go/src/github.com/hashicorp/consul-template
COPY . /go/src/github.com/hashicorp/consul-template
WORKDIR /go/src/github.com/hashicorp/consul-template

RUN make bootstrap updatedeps

ENTRYPOINT ["/usr/bin/make"]

CMD ["test"]
