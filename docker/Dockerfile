# This Dockerfile creates a production release image for the project. This
# downloads the release from releases.hashicorp.com and therefore requires that
# the release is published before building the Docker image.
#
# We don't rebuild the software because we want the exact checksums and
# binary signatures to match the software and our builds aren't fully
# reproducible currently.
FROM alpine:3

# NAME and VERSION are the name of the software in releases.hashicorp.com
# and the version to download.
ARG NAME=consul-template
ARG VERSION

# version label is required for build process
LABEL maintainer="John Eikenberry <jae@zhar.net>"
LABEL version=$VERSION

# UID and GID of consul-template user and group.
# These are the defaults, this makes them explicit and overridable.
ARG UID=100
ARG GID=1000

# Set ARGs as ENV so that they can be used in ENTRYPOINT/CMD
ENV NAME=${NAME}
ENV VERSION=${VERSION}

# This is the location of the releases.
ENV HASHICORP_RELEASES=https://releases.hashicorp.com

# Create a user and group first so the IDs get set the same way,
# even as the rest of this may change over time.
RUN addgroup -g ${GID} ${NAME} && adduser -u ${UID} -S -G ${NAME} ${NAME}

# Set up certificates, base tools, and software.
COPY fetch-n-verify.sh docker-entrypoint.sh /
RUN /fetch-n-verify.sh # removes self when done

ENTRYPOINT ["/docker-entrypoint.sh"]

USER ${NAME}:${NAME}
CMD /bin/${NAME}

