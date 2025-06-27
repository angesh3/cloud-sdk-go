FROM golang:1.21-alpine AS build

WORKDIR /app

# Install git for go get
RUN apk add --no-cache git

# Copy SDK
COPY . /sdk-source

# Copy frontend and backend code
COPY pxgrid-cloud-demo/backend /app/backend
COPY pxgrid-cloud-demo/frontend /app/frontend
COPY pxgrid-cloud-demo/config /app/config

WORKDIR /app/backend

# Initialize module and get dependencies
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o pxgrid-demo

# Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy the binary from build stage
COPY --from=build /app/backend/pxgrid-demo .

# Copy frontend for serving
COPY --from=build /app/frontend /app/frontend

# Copy config directory
COPY --from=build /app/config /app/config

# Install CA certificates for TLS
RUN apk --no-cache add ca-certificates

# Set executable permissions
RUN chmod +x /app/pxgrid-demo

# Expose port for API and WebSocket
EXPOSE 8080

# Run the application
CMD ["/app/pxgrid-demo"]
