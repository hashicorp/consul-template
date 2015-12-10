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
	go list $(TEST) | xargs -n1 go test -timeout=60s -parallel=4 $(TESTARGS)

# testrace runs the race checker
testrace: generate
	go list $(TEST) | xargs -n1 go test -race $(TESTARGS)

# updatedeps installs all the dependencies Consul Template needs to run and
# build
updatedeps:
	go get -u github.com/mitchellh/gox
	go list ./... \
		| xargs go list -f '{{ join .Deps "\n" }}{{ printf "\n" }}{{ join .TestImports "\n" }}' \
		| grep -v github.com/hashicorp/$(NAME) \
		| xargs go get -f -u -v

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	find . -type f -name '.DS_Store' -delete
	go generate ./...

.PHONY: default bin dev dist test testrace updatedeps generate
