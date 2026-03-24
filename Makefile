build:
	@go build -o bin/fs ./cmd/distributed-fs

build-metadata:
	@go build -o bin/metadata ./cmd/metadata

run: build
	@./bin/fs

run-metadata: build-metadata
	@./bin/metadata

test:
	@go test ./...