# syntax=docker/dockerfile:1
# Kulee — multi-stage Docker build.
# Stage 1: build the Go binary
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache nodejs npm
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /build/kulee ./cmd/server
RUN cd web && npm ci && npm run build

# Stage 2: minimal runtime image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/kulee /kulee
COPY --from=builder /build/web/dist /web/dist
EXPOSE 8080
CMD ["/kulee"]