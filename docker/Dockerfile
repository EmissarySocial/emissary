FROM golang:1.21

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# For now, use config-docker.json in the source as our config.json
COPY docker/config-docker.json ./config.json

# Build the Go app
RUN go build -o emissary server.go

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./emissary"]
