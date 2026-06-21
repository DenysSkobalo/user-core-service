.PHONY: build run test docker-up docker-down clean fmt lint install-lint

build:
	go build -v ./...

run:
	go run cmd/api/main.go

test:
	go test -v -race ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	go clean
	rm -rf bin/

fmt:
	golangci-lint fmt

install-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2

lint:
	golangci-lint run ./...
