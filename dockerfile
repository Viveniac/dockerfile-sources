# Stage 1: Build the Go binary
FROM golang:1.21 as builder

# Set working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to leverage Docker caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the binary with CGO disabled for a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o dockerfile-sources main.go

# Stage 2: Create the final minimal runtime image
FROM alpine:latest

# Install certificates and Git (Git is needed for cloning repositories)
RUN apk --no-cache add ca-certificates git

# Set working directory in the final image
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/dockerfile-sources .

# Set the entrypoint so the container runs our tool
ENTRYPOINT ["./dockerfile-sources"]
