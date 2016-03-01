TEST?=./...
NAME?=$(shell basename "$(CURDIR)")
VERSION = $(shell awk -F\" '/^const Version/ { print $$2; exit }' main.go)

default: test

# bin generates the releasable binaries
bin:
	@sh -c "'$(CURDIR)/scripts/build.sh'"

# dev creates binaries for testing locally. There are put into ./bin and $GOPATH
dev:
	@DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# dist creates the binaries for distibution
dist: bin
	@sh -c "'$(CURDIR)/scripts/dist.sh' $(VERSION)"

# test runs the test suite and vets the code
test: generate
	go list $(TEST) | grep -v vendor | xargs -n1 go test -timeout=60s -parallel=4 $(TESTARGS)

# testrace runs the race checker
testrace: generate
	go list $(TEST) | grep -v vendor | xargs -n1 go test -race $(TESTARGS)

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	find . -type f -name '.DS_Store' -delete
	go list $(TEST) | grep -v vendor | go generate $(TESTARGS)

.PHONY: default bin dev dist test testrace generate
