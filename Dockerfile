FROM oven/bun:latest

WORKDIR /app

COPY package.json bun.lock ./
RUN bun install

COPY . .


ENTRYPOINT ["bun", "src/index.ts"]
