projectname?=cligram

# --- JS Backend Variables ---
JS_BACKEND_BINARY_DIR=internal/assets/resources
JS_BACKEND_BINARY=$(JS_BACKEND_BINARY_DIR)/cligram-js-backend
JS_BUILD_OUTPUT=js/bin/cligram-js

# --- Dynamically find all relevant source and config files ---
# This creates a list of all .ts and .json files in js/src, plus key config files.
# The -o flag in find means OR.
JS_SOURCES := $(shell find js/src -type f \( -name '*.ts' -o -name '*.json' \)) \
              js/package.json \
              js/bun.lock \
              js/tsconfig.json

default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: $(JS_BACKEND_BINARY) ## build golang binary with embedded JS backend
	@echo "--> Building Go application..."
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -o $(projectname)

# Target to prepare the JS binary for embedding
$(JS_BACKEND_BINARY): $(JS_BUILD_OUTPUT)
	@echo "--> Preparing JS backend for embedding..."
	@mkdir -p $(JS_BACKEND_BINARY_DIR)
	@cp $(JS_BUILD_OUTPUT) $(JS_BACKEND_BINARY)

# Target to build the JS source using bun
# It now depends on the specific .ts and .json files in js/src.
$(JS_BUILD_OUTPUT): $(JS_SOURCES)
	@echo "--> Building standalone JS binary with bun (source/config changed)..."
	@cd js && bun install && bun run build

.PHONY: install
install: ## install golang binary
	@go install -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"

.PHONY: run
run: build ## build and run the app
	@./$(projectname)

.PHONY: bootstrap
bootstrap: ## install build deps
	go generate -tags tools tools/tools.go

.PHONY: test
test: clean ## display test coverage
	go test --cover -parallel=1 -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | sort -rnk3

.PHONY: clean
clean: ## clean up environment
	@echo "--> Cleaning up..."
	@rm -rf coverage.out dist/ $(projectname)
	@rm -rf embed
	@rm -rf js/bin
	@rm -rf js/node_modules

.PHONY: cover
cover: ## display test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: fmt
fmt: ## format go files
	gofumpt -w .
	gci write .

.PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml

.PHONY: pre-commit
pre-commit:	## run pre-commit hooks
	pre-commit run --all-files