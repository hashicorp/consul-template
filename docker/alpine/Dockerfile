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
FROM alpine:latest
LABEL maintainer "John Eikenberry <jae@zhar.net>"

# UID and GID of consul-template user and group.
# These are the defaults, this makes them explicit and overridable.
ARG UID=100
ARG GID=1000

# Create a consul-template user and group first so the IDs get set the same way,
# even as the rest of this may change over time.
RUN addgroup -g ${GID} consul-template && \
    adduser -u ${UID} -S -G consul-template consul-template

# Set up certificates, tini (formerly dumb-init), and Consul Template (CT).
RUN apk add --no-cache ca-certificates tini && \
    update-ca-certificates --fresh

# Install consul-template
COPY --from=builder "/consul-template" "/bin/consul-template"

# The agent will be started with /consul-template/config as the configuration
# directory so you can add additional config files in that location.
RUN mkdir -p /consul-template/data && \
    mkdir -p /consul-template/config && \
    chown -R consul-template:consul-template /consul-template

# Expose the consul-template data directory as a volume since that's where
# shared results should be rendered.
VOLUME /consul-template/data

# The entry point script uses dumb-init as the top-level process to reap any
# zombie processes created by Consul Template sub-processes.
COPY "docker/alpine/docker-entrypoint.sh" "/bin/docker-entrypoint.sh"
ENTRYPOINT ["/bin/docker-entrypoint.sh"]
USER ${UID}:${GID}

# Run consul-template by default
CMD ["/bin/consul-template"]
