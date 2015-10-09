TEST?=./...
VERSION = $(shell awk -F\" '/^const Version/ { print $$2 }' main.go)

default: test

# bin generates the releasable binaries for Consul Template
bin: generate
	@sh -c "'$(CURDIR)/scripts/build.sh'"

# dev creates binares for testing locally. There are put into ./bin and $GOPAHT
dev: generate
	@CT_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# dist creates the binaries for distibution
dist: bin
	./scripts/dist.sh $(VERSION)

# test runs the test suite and vets the code
test: generate
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4
	@$(MAKE) vet

# testrace runs the race checker
testrace: generate
	go test -race $(TEST) $(TESTARGS)

# updatedeps instlals all the dependencies Consul Template needs to run and
# build
updatedeps:
	go get -u github.com/mitchellh/gox
	go list -f '{{range .TestImports}}{{.}} {{end}}' ./... \
		| xargs go list -f '{{join .Deps "\n"}}' \
		| grep -v github.com/hashicorp/consul-template \
		| sort -u \
		| xargs go get -f -u -v

# vet runs Go's vetter and reports any common errors
vet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@echo "go tool vet $(VETARGS) ."
	@go tool vet $(VETARGS) . ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for reviewal."; \
	fi

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	find . -type f -name '.DS_Store' -delete
	go generate ./...

.PHONY: default bin dev dist test testrace updatedeps vet generate
