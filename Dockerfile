# Build stage for Node.js and Go
FROM  oven/bun:1.0.29 AS builder

# Architecture-specific variables
ARG TARGETARCH
ARG BUILDPLATFORM
ARG TARGETOS

# Install Go 1.24
RUN apt-get update && apt-get install -y \
    wget \
    git \
    make \
    && if [ "$TARGETARCH" = "arm64" ]; then \
       wget https://go.dev/dl/go1.24.4.linux-arm64.tar.gz -O go.tar.gz; \
    else \
       wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz -O go.tar.gz; \
    fi \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Set the working directory
WORKDIR /app

# Copy js package files first for better caching
COPY js/package.json js/bun.lock js/

# Install JS dependencies
WORKDIR /app/js
RUN bun install

# Copy the rest of the application
WORKDIR /app
COPY . .

# Build the JS backend
WORKDIR /app/js
RUN bun run build

# Create the directory for the JS backend binary
WORKDIR /app
RUN mkdir -p internal/assets/resources

# Copy the JS binary to the location expected by the Go build
RUN cp js/bin/cligram-js internal/assets/resources/cligram-js-backend

# Now build the Go binary with the embedded JS backend
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.version=$(git describe --abbrev=0 --tags || echo dev)" -o cligram

# Final stage
FROM debian:stable-slim

# Install runtime dependencies if any
RUN apt-get update && apt-get install -y ca-certificates

# Copy only the final binary
COPY --from=builder /app/cligram /usr/bin/cligram

# Create necessary directories
# Create cache directories for both config and JS backend
RUN mkdir -p /root/.cache/cligram /root/.cligram

# Pre-install the JS backend binary to the cache location
COPY --from=builder /app/internal/assets/resources/cligram-js-backend /root/.cache/cligram/cligram-js-backend

# Set proper permissions to allow for removal/replacement of the binary
RUN chmod +x /root/.cache/cligram/cligram-js-backend

# Verify permissions
RUN ls -l /root/.cache/cligram/cligram-js-backend

# Set the entrypoint
ENTRYPOINT ["sh"]