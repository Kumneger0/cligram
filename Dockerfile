FROM oven/bun:latest

# Accept build arguments
ARG TELEGRAM_API_ID
ARG TELEGRAM_API_HASH

# Set environment variables inside the image
ENV TELEGRAM_API_ID=$TELEGRAM_API_ID
ENV TELEGRAM_API_HASH=$TELEGRAM_API_HASH

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install

COPY . .


CMD ["bun", "src/index.ts"]
