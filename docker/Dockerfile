FROM golang:1.21-bookworm as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o emissary server.go

# Start a new stage from scratch
FROM debian:bookworm-slim

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/emissary .
COPY --from=builder /app/docker/config-entrypoint.sh .

# Command to run the executable
CMD ["./emissary"]
