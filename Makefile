.DEFAULT_GOAL := check

.PHONY: check
check: vet lint test ## Run every gate CI runs

BIN_DIR := bin

.PHONY: build
build: ## Compile all packages and write bin/linkforge
	go build ./...
	cd pkg && go build ./...
	go build -o $(BIN_DIR)/linkforge ./cmd/linkforge

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

.PHONY: test
test: ## Run tests with the race detector
	go test -race ./...
	cd pkg && go test -race ./...

.PHONY: cover
cover: ## Report test coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: bench
bench: ## Run benchmarks
	cd pkg && go test -run '^$$' -bench . -benchmem ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...
	cd pkg && go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (install: brew install golangci-lint)
	golangci-lint run
	cd pkg && golangci-lint run

.PHONY: tidy
tidy: ## Tidy every module in the workspace
	go mod tidy
	cd pkg && go mod tidy
	go work sync

.PHONY: help
help: ## List targets
	@grep -hE '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  %-8s %s\n", $$1, $$2}'
