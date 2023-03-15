GOPATH ?= $(shell go env GOPATH)
BIN_DIR := $(GOPATH)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

IMG ?= iawia002/tower-extension:latest

.PHONY: lint test

lint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) v1.51.2

test:
	@go test ./... -coverprofile=coverage.out
	@go tool cover -func coverage.out | tail -n 1 | awk '{ print "total: " $$3 }'

apiserver:
	docker build -f build/apiserver/Dockerfile . -t ${IMG}

apiserver-push:
	docker buildx build --platform linux/amd64,linux/arm64 -f build/apiserver/Dockerfile . -t ${IMG} --push
