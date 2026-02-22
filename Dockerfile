# Build stage
FROM golang:1.25-alpine AS builder

# Version can be passed as build argument
ARG VERSION=dev

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Copy source code (needed for go mod tidy to resolve all dependencies)
COPY cmd/ cmd/
COPY pkg/ pkg/

# Download dependencies and populate go.sum with all transitive dependencies
RUN go mod download && go mod tidy

# Build the dnsrbl-exporter with version
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-w -s -X main.Version=${VERSION}" -o dnsrbl-exporter ./cmd/dnsrbl-exporter

# Runtime stage
FROM alpine:3.23.3

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /workspace/dnsrbl-exporter .

# Copy lists.txt
COPY lists.txt .

# Default mode is direct
ENTRYPOINT ["/app/dnsrbl-exporter"]

USER nobody

EXPOSE 8000
