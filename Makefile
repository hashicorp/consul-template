TEST?=./...
NAME?=$(shell awk -F\" '/^\tName/ { print $$2; exit }' version.go)
VERSIONMAJ?=$(shell awk -F\" '/^\tVersion/ { print $$2; exit }' version.go)
VERSIONPRE?=$(shell awk -F\" '/^\tVersionPrerelease/ { print $$2; exit }' version.go)
VERSION?=$(shell if [ "${VERSIONPRE}x" != x ]; then echo "${VERSIONMAJ}-${VERSIONPRE}"; else echo "${VERSIONMAJ}"; fi)
EXTERNAL_TOOLS=\
	github.com/mitchellh/gox
BUILD_TAGS?=vault

default: test

# bin generates the binaries for all platforms.
bin: generate
	@BUILD_TAGS='${BUILD_TAGS}' sh -c "'${CURDIR}/scripts/build.sh' '${NAME}'"

# dev creates binares for testing locally - they are put into ./bin and $GOPATH.
dev: generate
	@BUILD_TAGS='${BUILD_TAGS}' DEV=1 sh -c "'${CURDIR}/scripts/build.sh' '${NAME}'"

# dist creates the binaries for distibution.
dist:
	@BUILD_TAGS='${BUILD_TAGS}' sh -c "'${CURDIR}/scripts/dist.sh' '${NAME}' '${VERSION}'"

# test runs the test suite and vets the code.
test: generate
	@echo "==> Running tests..."
	@go list $(TEST) \
		| grep -v "/vendor/" \
		| xargs -n1 go test -tags='${BUILD_TAGS}' -timeout=60s -parallel=10 ${TESTARGS}

# testrace runs the race checker
testrace: generate
	@echo "==> Running tests (race)..."
	@go list $(TEST) \
		| grep -v "/vendor/" \
		| xargs -n1 go test -tags='${BUILD_TAGS}' -timeout=60s -race ${TESTARGS}

# updatedeps installs all the dependencies needed to run and build.
updatedeps:
	@sh -c "'${CURDIR}/scripts/deps.sh' '${NAME}'"

# generate runs `go generate` to build the dynamically generated source files.
generate:
	@echo "==> Generating..."
	@find . -type f -name '.DS_Store' -delete
	@go list ./... \
		| grep -v "/vendor/" \
		| xargs -n1 go generate -tags='${BUILDTAGS}'

# bootstrap installs the necessary go tools for development/build.
bootstrap:
	@echo "==> Bootstrapping..."
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing "$$t"..." ; \
		go get -u "$$t"; \
	done

.PHONY: default bin dev dist test testrace updatedeps vet generate bootstrap
