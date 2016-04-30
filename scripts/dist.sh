#!/usr/bin/env bash
set -e

# Get the name from the command line.
NAME=$1
if [ -z $NAME ]; then
  echo "Please specify a name."
  exit 1
fi

# Get the version from the command line.
VERSION=$2
if [ -z $VERSION ]; then
  echo "Please specify a version."
  exit 1
fi

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"

# Change into that dir because we expect that.
cd "$DIR"

# Build
if [ -z $NOBUILD ]; then
  echo "==> Building binaries in container..."
  docker run \
    --rm \
    --workdir="/go/src/github.com/hashicorp/${NAME}" \
    --volume="$(pwd):/go/src/github.com/hashicorp/${NAME}" \
    golang:1.6.2 /bin/sh -c "make bootstrap && make bin"
fi

# Generate the tag.
if [ -z $NOTAG ]; then
  echo "==> Tagging..."
  git commit --allow-empty -a --gpg-sign=348FFC4C -m "Release v$VERSION"
  git tag -a -m "Version $VERSION" -s -u 348FFC4C "v${VERSION}" master
fi

# Zip all the files.
echo "==> Packaging..."
for PLATFORM in $(find ./pkg -mindepth 1 -maxdepth 1 -type d); do
  OSARCH=$(basename ${PLATFORM})
  echo "--> ${OSARCH}"

  pushd $PLATFORM >/dev/null 2>&1
  zip ../${OSARCH}.zip ./*
  popd >/dev/null 2>&1
done

# Move everything into dist.
rm -rf ./pkg/dist
mkdir -p ./pkg/dist
for FILENAME in $(find ./pkg -mindepth 1 -maxdepth 1 -type f); do
  FILENAME=$(basename $FILENAME)
  cp ./pkg/${FILENAME} ./pkg/dist/${NAME}_${VERSION}_${FILENAME}
done

# Make and sign the checksums.
pushd ./pkg/dist
shasum -a256 * > ./${NAME}_${VERSION}_SHA256SUMS
if [ -z $NOSIGN ]; then
  echo "==> Signing..."
  gpg --default-key 348FFC4C --detach-sig ./${NAME}_${VERSION}_SHA256SUMS
fi
popd
