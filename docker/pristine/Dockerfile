### Used for pristine builds
#
ARG GOVERSION=latest
FROM golang:${GOVERSION}
LABEL maintainer "John Eikenberry <jae@zhar.net>"

RUN apt-get update && \
    apt-get -y install gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu
