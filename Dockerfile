
FROM  oven/bun:1.2.18 AS builder

ARG TARGETARCH
ARG BUILDPLATFORM
ARG TARGETOS

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

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /app

COPY js/package.json js/bun.lock js/

WORKDIR /app/js
RUN bun install

WORKDIR /app
COPY . .

WORKDIR /app/js
RUN bun run build

WORKDIR /app
RUN mkdir -p internal/assets/resources

RUN cp js/bin/cligram-js internal/assets/resources/cligram-js-backend

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-X main.version=$(git describe --abbrev=0 --tags || echo dev)" -o cligram

FROM debian:stable-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=builder /app/cligram /usr/bin/cligram

RUN mkdir -p /root/.cache/cligram /root/.cligram

COPY --from=builder /app/internal/assets/resources/cligram-js-backend /root/.cache/cligram/cligram-js-backend

RUN chmod +x /root/.cache/cligram/cligram-js-backend

RUN ls -l /root/.cache/cligram/cligram-js-backend

ENTRYPOINT ["/usr/bin/cligram"]