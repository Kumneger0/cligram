FROM oven/bun:latest

# build arguments
ARG TELEGRAM_API_ID
ARG TELEGRAM_API_HASH

# set environment variables inside the image
ENV TELEGRAM_API_ID=$TELEGRAM_API_ID
ENV TELEGRAM_API_HASH=$TELEGRAM_API_HASH

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install

COPY . .


ENTRYPOINT ["bun", "src/index.ts"]
