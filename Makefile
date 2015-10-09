TEST?=./...
NAME = $(shell awk -F\" '/^const Name/ { print $$2; exit }' main.go)
VERSION = $(shell awk -F\" '/^const Version/ { print $$2; exit }' main.go)

default: test

# bin generates the releasable binaries for Consul Template
bin: generate
	@sh -c "'$(CURDIR)/scripts/build.sh'"

# dev creates binares for testing locally. There are put into ./bin and $GOPAHT
dev: generate
	@CT_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# dist creates the binaries for distibution
dist: bin
	@sh -c "'$(CURDIR)/scripts/dist.sh' $(VERSION)"

# test runs the test suite and vets the code
test: generate
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4

# testrace runs the race checker
testrace: generate
	go test -race $(TEST) $(TESTARGS)

# updatedeps installs all the dependencies Consul Template needs to run and
# build
updatedeps:
	go get -u github.com/mitchellh/gox
	go get -f -t -u ./...
	go list ./... \
		| xargs go list -f '{{join .Deps "\n"}}' \
		| grep -v github.com/hashicorp/consul-template \
		| grep -v '/internal/' \
		| sort -u \
		| xargs go get -f -u

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	find . -type f -name '.DS_Store' -delete
	go generate ./...

.PHONY: default bin dev dist test testrace updatedeps vet generate
