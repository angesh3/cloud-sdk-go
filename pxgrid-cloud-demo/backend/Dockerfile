FROM golang:1.21-alpine AS build

WORKDIR /app

# Copy go.mod and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o pxgrid-demo

# Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy the binary from build stage
COPY --from=build /app/pxgrid-demo .
COPY --from=build /app/config /app/config

# Copy frontend files
COPY --from=build /app/frontend /app/frontend

# Install CA certificates for TLS
RUN apk --no-cache add ca-certificates

# Set executable permissions
RUN chmod +x /app/pxgrid-demo

# Expose port for API and WebSocket
EXPOSE 8080

# Run the application
CMD ["/app/pxgrid-demo"]
