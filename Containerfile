
# Stage 1: Build the Go application
FROM golang:1.27-alpine AS builder

WORKDIR /src

# Copy Go module definition
COPY go.mod ./

# Copy application source
COPY app/ ./app/

# Compile a lightweight Linux executable
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/opspulse ./app

# Stage 2: Create the runtime image
FROM alpine:3.22

WORKDIR /app

# Copy only the compiled application
COPY --from=builder /out/opspulse ./opspulse

# Application configuration
ENV APP_NAME=OpsPulse
ENV APP_VERSION=1.0.0
ENV APP_ENV=production
ENV APP_PORT=8080

# Document the application port
EXPOSE 8080

# Start the application
ENTRYPOINT ["./opspulse"]
