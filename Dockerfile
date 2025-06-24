# Build stage for Node.js and Go
FROM oven/bun:1.2.5 AS builder

# Install Go 1.24
RUN apt-get update && apt-get install -y \
    wget \
    git \
    make \
    && wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz \
    && rm go1.24.4.linux-amd64.tar.gz \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Set the working directory
WORKDIR /app

# Copy the entire project
COPY . .

# First build the JS backend
RUN cd js && bun install && bun run build

# Create the directory for the JS backend binary
RUN mkdir -p internal/assets/resources

# Copy the JS binary to the location expected by the Go build
RUN cp js/bin/cligram-js internal/assets/resources/cligram-js-backend

# Now build the Go binary with the embedded JS backend
RUN go build -ldflags "-X main.version=$(git describe --abbrev=0 --tags || echo dev)" -o cligram

# Final stage
FROM alpine:3.22

# Install runtime dependencies if any
RUN apk add --no-cache ca-certificates

# Copy only the final binary
COPY --from=builder /app/cligram /usr/bin/cligram

# Create necessary directories
# Create cache directories for both config and JS backend
RUN mkdir -p /root/.cache/cligram /root/.cligram

# Pre-install the JS backend binary to the cache location
COPY --from=builder /app/internal/assets/resources/cligram-js-backend /root/.cache/cligram/cligram-js-backend

# Set proper permissions to allow for removal/replacement of the binary
RUN chmod 755 /root/.cache/cligram/cligram-js-backend

# Set the entrypoint
ENTRYPOINT ["/usr/bin/cligram"]