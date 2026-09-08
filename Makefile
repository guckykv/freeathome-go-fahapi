.PHONY: check build test cover vet fmt fmt-check clean

# The one command to run before committing.
check: fmt-check vet build test

build:
	go build ./...

# -race is the point: it covers the concurrency the websocket loop relies on.
test:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	@echo "detail: go tool cover -html=coverage.out"

# go vet caches results and can report a stale pass. Clear it first.
vet:
	@go clean -cache
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt'ed:"; echo "$$unformatted"; exit 1; \
	fi

clean:
	rm -f coverage.out
