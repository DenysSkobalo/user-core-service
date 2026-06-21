FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go build -o user-core-service ./cmd/api

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/user-core-service .

EXPOSE 8080

CMD ["./user-core-service"]
