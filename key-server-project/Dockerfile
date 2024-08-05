# Use the official Go image as the base image
FROM golang:1.21-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire project directory to the working directory
COPY . .

# Build the Go application
RUN go build -o key-server .

# Create a smaller final image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/key-server /app/key-server

# Expose the port the server will run on
EXPOSE 1123

# Define the command to run the application
CMD ["./key-server"]