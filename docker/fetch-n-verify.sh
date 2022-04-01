#!/bin/sh

# requires environment variables..
#
# HASHICORP_RELEASES - URL for releases
# NAME - application's name
# VERSION - application version (eg. 1.2.3)

set -eux
apk add --no-cache ca-certificates gnupg

BUILD_GPGKEY=C874011F0AB405110D02105534365D9472D7468F
found=''

for server in \
    hkp://p80.pool.sks-keyservers.net:80 \
    hkp://keyserver.ubuntu.com:80 \
    hkp://pgp.mit.edu:80 \
; do
    echo "Fetching GPG key $BUILD_GPGKEY from $server";
    gpg --keyserver "$server" --recv-keys "$BUILD_GPGKEY" && found=yes && break;
done

test -z "$found" && echo >&2 "error: failed to fetch GPG key $BUILD_GPGKEY" && exit 1
mkdir -p /tmp/build && cd /tmp/build

apkArch="$(apk --print-arch)"
case "${apkArch}" in \
    aarch64) ARCH='arm64' ;;
    armhf) ARCH='arm' ;;
    x86) ARCH='386' ;;
    x86_64) ARCH='amd64' ;;
    *) echo >&2 "error: unsupported architecture: ${apkArch} (see ${HASHICORP_RELEASES}/${NAME}/${VERSION}/)" && exit 1 ;;
esac

wget ${HASHICORP_RELEASES}/${NAME}/${VERSION}/${NAME}_${VERSION}_linux_${ARCH}.zip
wget ${HASHICORP_RELEASES}/${NAME}/${VERSION}/${NAME}_${VERSION}_SHA256SUMS
wget ${HASHICORP_RELEASES}/${NAME}/${VERSION}/${NAME}_${VERSION}_SHA256SUMS.sig
gpg --batch --verify ${NAME}_${VERSION}_SHA256SUMS.sig ${NAME}_${VERSION}_SHA256SUMS
grep ${NAME}_${VERSION}_linux_${ARCH}.zip ${NAME}_${VERSION}_SHA256SUMS | sha256sum -c
unzip -d /bin ${NAME}_${VERSION}_linux_${ARCH}.zip

apk del gnupg
cd /tmp
rm -rf /tmp/build
gpgconf --kill gpg-agent || true
rm -rf /root/.gnupg || true
rm -f "$0"
