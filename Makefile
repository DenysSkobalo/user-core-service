.PHONY: build run test docker-up docker-down clean

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
