projectname?=cligram

default: help

.PHONY: help
help: 
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: 
	@echo "--> Building Go application..."
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags) -X main.telegramAPIID=$(shell echo $$TELEGRAM_API_ID) -X main.telegramAPIHash=$(shell echo $$TELEGRAM_API_HASH)" -o $(projectname)

.PHONY: install
install: build 
	@echo "--> Installing cligram to /usr/local/bin..."
	@sudo cp $(projectname) /usr/local/bin/
	@echo "--> Installation complete. Run 'cligram' to start."

.PHONY: run
run: build 
	@./$(projectname)

.PHONY: bootstrap
bootstrap: 
	go generate -tags tools tools/tools.go

.PHONY: test
test: clean
	go test --cover -parallel=1 -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | sort -rnk3

.PHONY: clean
clean: 
	@echo "--> Cleaning up..."
	@rm -rf coverage.out dist/ $(projectname)
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