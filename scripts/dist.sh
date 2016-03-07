#!/bin/bash
set -e

# Get the version from the command line
VERSION=$1
if [ -z $VERSION ]; then
    echo "Please specify a version."
    exit 1
fi

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that
cd $DIR

# Generate and save godeps
if [ -z $NODEPS ]; then
  rm -rf Godeps/
  rm -rf vendor/
  godep save ./...
  git add -f vendor/
  git add -f Godeps/
  git commit -q -a -m "Add godeps for $VERSION"
fi

# Generate the tag
if [ -z $NOTAG ]; then
  echo "==> Tagging..."
  git commit --allow-empty -a --gpg-sign=348FFC4C -m "Release v$VERSION"
  git tag -a -m "Version $VERSION" -s -u 348FFC4C "v${VERSION}" master
fi

if [ -z $NOBUILD ]; then
  echo "==> Building..."
  docker run \
    --rm \
    --workdir="/go/src/github.com/hashicorp/consul-template" \
    --volume="$(pwd):/go/src/github.com/hashicorp/consul-template" \
    golang:1.6.0 "make updatedeps && make bin"
fi

# Zip all the files
echo "==> Packaging..."
for PLATFORM in $(find ./pkg -mindepth 1 -maxdepth 1 -type d); do
  OSARCH=$(basename ${PLATFORM})
  echo "--> ${OSARCH}"

  pushd $PLATFORM >/dev/null 2>&1
  zip ../${OSARCH}.zip ./*
  popd >/dev/null 2>&1
done

# Move everything into dist
rm -rf ./pkg/dist
mkdir -p ./pkg/dist
for FILENAME in $(find ./pkg -mindepth 1 -maxdepth 1 -type f); do
    FILENAME=$(basename $FILENAME)
    cp ./pkg/${FILENAME} ./pkg/dist/consul-template_${VERSION}_${FILENAME}
done

# Make the checksums
pushd ./pkg/dist
shasum -a256 * > ./consul-template_${VERSION}_SHA256SUMS
if [ -z $NOSIGN ]; then
  echo "==> Signing..."
  gpg --default-key 348FFC4C --detach-sig ./consul-template_${VERSION}_SHA256SUMS
fi
popd

# Remove godeps now that we are done
if [ -z $NODEPS ]; then
  rm -rf Godeps/
  rm -rf vendor/
fi
