projectname?=cligram

default: help

.PHONY: help
help: 
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build cligram binary with embedded call-helper
	@echo "--> Building unified cligram application..."
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo dev) -X main.telegramAPIID=$(shell echo $$TELEGRAM_API_ID) -X main.telegramAPIHash=$(shell echo $$TELEGRAM_API_HASH)" -o $(projectname)

.PHONY: install
install: build ## Install cligram to /usr/local/bin
	@echo "--> Installing cligram to /usr/local/bin..."
	@sudo cp $(projectname) /usr/local/bin/
	@echo "--> Installation complete. Run 'cligram' to start."

.PHONY: run
run: build 
	@./$(projectname)

.PHONY: bootstrap
bootstrap: 
	go generate -tags tools tools/tools.go

.PHONY: test-core
test-core: ## Run tests for core packages
	@echo "--> Running core tests..."
	@go test --cover -parallel=1 -v -coverprofile=coverage.out ./internal/...
	@go tool cover -func=coverage.out | sort -rnk3

.PHONY: test-call-helper
test-call-helper: ## Run tests for Cgo call-helper worker (requires libntgcalls)
	@echo "--> Running call-helper worker tests..."
	@go test -v ./cmd/callhelper/...

.PHONY: test
test: clean test-core test-call-helper ## Run test suite for core application

.PHONY: clean
clean: 
	@echo "--> Cleaning up..."
	@rm -rf coverage.out dist/ $(projectname) cligram-voip
	@rm -rf embed
	@rm -rf js/bin
	@rm -rf js/node_modules

.PHONY: cover
cover: 
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: fmt
fmt: 
	gofumpt -w .
	gci write .

.PHONY: lint
lint: 
	golangci-lint run -c .golangci.yml

.PHONY: pre-commit
pre-commit:
	pre-commit run --all-files

.PHONY: hooks
hooks: 
	@chmod +x scripts/hooks/commit-msg
	@git config core.hooksPath scripts/hooks
	@echo "--> Git hooks installed (commit-msg)."