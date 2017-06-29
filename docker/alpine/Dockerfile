FROM alpine:latest
LABEL maintainer "Seth Vargo <seth@sethvargo.com> (@sethvargo)"

# This is the release of https://github.com/hashicorp/docker-base to pull in order
# to provide HashiCorp-built versions of basic utilities like dumb-init and gosu.
ENV DOCKER_BASE_VERSION=0.0.4

# This is the location of the releases.
ENV HASHICORP_RELEASES=https://releases.hashicorp.com

# Create a consul-template user and group first so the IDs get set the same way,
# even as the rest of this may change over time.
RUN addgroup consul-template && \
    adduser -S -G consul-template consul-template

# Set up certificates, our base tools, and Consul Template (CT).
RUN apk add --no-cache ca-certificates curl gnupg libcap openssl && \
    gpg --keyserver pgp.mit.edu --recv-keys 91A6E7F85D05C65630BEF18951852D87348FFC4C && \
    mkdir -p /tmp/build && \
    cd /tmp/build && \
    wget ${HASHICORP_RELEASES}/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_linux_amd64.zip && \
    wget ${HASHICORP_RELEASES}/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS && \
    wget ${HASHICORP_RELEASES}/docker-base/${DOCKER_BASE_VERSION}/docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS.sig && \
    gpg --batch --verify docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS.sig docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS && \
    grep ${DOCKER_BASE_VERSION}_linux_amd64.zip docker-base_${DOCKER_BASE_VERSION}_SHA256SUMS | sha256sum -c && \
    unzip docker-base_${DOCKER_BASE_VERSION}_linux_amd64.zip && \
    cp bin/gosu bin/dumb-init /bin && \
    cd /tmp && \
    rm -rf /tmp/build && \
    apk del gnupg openssl && \
    rm -rf /root/.gnupg

# Install consul-template
ADD "./pkg/linux_amd64/consul-template" "/bin/consul-template"

# The agent will be started with /consul-template/config as the configuration directory
# so you can add additional config files in that location.
RUN mkdir -p /consul-template/data && \
    mkdir -p /consul-template/config && \
    chown -R consul-template:consul-template /consul-template

# Expose the consul-template data directory as a volume since that's where
# shared results should be rendered.
VOLUME /consul-template/data

# The entry point script uses dumb-init as the top-level process to reap any
# zombie processes created by Consul Template sub-processes.
COPY docker/alpine/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
ENTRYPOINT ["docker-entrypoint.sh"]

# Run consul-template by default
CMD ["/bin/consul-template"]
