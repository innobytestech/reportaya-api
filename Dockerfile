# Stage 1: Builder
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Cached if go.mod and go.sum are unchanged.
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app.
# CGO_ENABLED=0 is important for a static binary (required for distroless).
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Stage 2: Runner
FROM gcr.io/distroless/static-debian12
WORKDIR /

# Copy the binary
COPY --from=builder /app/main .

# Copy migrations preserving the internal path expected at runtime.
COPY --from=builder /app/internal/persistence/migrations ./internal/persistence/migrations

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/main"]
