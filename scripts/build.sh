#!/usr/bin/env bash
#
# This script builds the application from source for multiple platforms.
set -e

# Get the name from the command line
NAME=$1
if [ -z $NAME ]; then
  echo "Please specify a name."
  exit 1
fi

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"

# Change into that directory
cd "$DIR"

# Get the git commit
GIT_COMMIT=$(git rev-parse HEAD)
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-"darwin freebsd linux netbsd openbsd solaris windows"}
XC_EXCLUDE=${XC_EXCLUDE:-"!darwin/arm"}

# Delete the old dir
echo "==> Removing old builds..."
rm -f bin/*
rm -rf pkg/*
mkdir -p bin/

# If its dev mode, only build for ourself
if [ "${DEV}x" != "x" ]; then
  XC_OS=$(go env GOOS)
  XC_ARCH=$(go env GOARCH)
fi

# Build!
echo "==> Building..."
export CGO_ENABLED=0
gox \
  -os="${XC_OS}" \
  -arch="${XC_ARCH}" \
  -osarch="${XC_EXCLUDE}" \
  -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY}" \
  -output "pkg/{{.OS}}_{{.Arch}}/${NAME}" \
  .

# Move all the compiled things to the $GOPATH/bin
GOPATH=${GOPATH:-$(go env GOPATH)}
case $(uname) in
  CYGWIN*)
    GOPATH="$(cygpath $GOPATH)"
    ;;
esac
OLDIFS=$IFS
IFS=: MAIN_GOPATH=($GOPATH)
IFS=$OLDIFS

# Copy our OS/Arch to the bin/ directory
DEV_PLATFORM="./pkg/$(go env GOOS)_$(go env GOARCH)"
for F in $(find ${DEV_PLATFORM} -mindepth 1 -maxdepth 1 -type f); do
  cp ${F} bin/
  cp ${F} ${MAIN_GOPATH}/bin/
done
