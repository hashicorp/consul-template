default: test

# test runs the test suite and vets the code
test: generate
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4

# testrace runs the race checker
testrace: generate
	go test -race $(TEST) $(TESTARGS)

# updatedeps installs all the dependencies gatedio needs to run and
# build
updatedeps:
	go get -f -t -u ./...
	go get -f -u ./...

# generate runs `go generate` to build the dynamically generated source files.
generate:
	find . -type f -name '.DS_Store' -delete
	go generate ./...

.PHONY: default bin dev dist test testrace updatedeps vet generate
