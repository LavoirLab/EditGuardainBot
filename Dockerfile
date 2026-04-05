FROM golang:1.25-bookworm AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o app .


FROM debian:bullseye-slim
WORKDIR /app

COPY --from=builder /etc/ssl/certs /etc/ssl/certs

COPY --from=builder /app/app .

ENTRYPOINT ["/app/app"]