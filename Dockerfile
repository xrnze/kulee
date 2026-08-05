# syntax=docker/dockerfile:1
# Kulee API — multi-stage Go build.
# Stage 1: build the Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /build/kulee ./cmd/server

# Stage 2: minimal runtime image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/kulee /kulee
EXPOSE 8080
CMD ["/kulee"]
