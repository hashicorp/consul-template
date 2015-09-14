#!/usr/bin/env bash
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that
cd $DIR

# Create a tarball for each package
echo "==> Packaging..."
pushd pkg > /dev/null
rm -rf ./dist
FILES=$(ls)
mkdir ./dist
for f in $FILES; do
  tar -zcf "dist/$f.tar.gz" "$f"
done
popd > /dev/null

# Make the checksums
if [ -z "$NOSIGN" ]; then
  echo "==> Signing..."
  pushd ./pkg/dist > /dev/null
  rm -f ./SHA256SUMS*
  shasum -a256 * > ./SHA256SUMS
  gpg --default-key 348FFC4C --detach-sig ./SHA256SUMS
  popd > /dev/null
fi

# All done
echo "==> Done!"
