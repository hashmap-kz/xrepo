# Variables
COV_REPORT 	:= coverage.txt
TEST_FLAGS 	:= -v -race -timeout 30s

# Lint the code
.PHONY: lint
lint:
	golangci-lint run --output.tab.path=stdout

# Run unit tests
.PHONY: test
test:
	go test -v -cover ./...

# Run tests with coverage
.PHONY: test-cov
test-cov:
	go test -coverprofile=$(COV_REPORT) ./...
	go tool cover -html=$(COV_REPORT)

######################################################################
# Integration tests
######################################################################

# Fast tests
.PHONY: test-integ
test-integ:
	@cd test/integration/environ && bash run.sh
	go test -tags=integration -v ./test/integration/... | tee test-integ-fast.log

.PHONY: teardown-integ
teardown-integ:
	@cd test/integration/environ && bash teardown.sh
