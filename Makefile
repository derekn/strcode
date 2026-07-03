VERSION = $(shell svu next 2>/dev/null)

.DEFAULT_GOAL := build
.PHONY: build clean lint test release update

clean:
	@rm -rf dist/

update:
	@go get -u ./...
	@go mod tidy

build:
	@goreleaser build --single-target --snapshot --clean

release:
	@git tag -f $(VERSION)
	@goreleaser release --clean

lint:
	go vet .
	-golangci-lint run
	gofmt -d .

test:
	@go test . --cover -v
