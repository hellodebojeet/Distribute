build:
	@go build -o bin/dfs ./cmd/cli

build-metadata:
	@go build -o bin/metadata ./cmd/metadata

run: build
	@./bin/dfs

run-metadata: build-metadata
	@./bin/metadata

test:
	@go test ./...