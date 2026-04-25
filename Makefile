.PHONY: test test-race lint tidy

test:
	go test ./...

test-race:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

lint:
	golangci-lint run

tidy:
	go mod tidy
