FROM oven/bun:latest

WORKDIR /app

# Copy package manifest files and install dependencies
COPY package.json bun.lock ./
RUN bun install

# Copy the rest of your source code
COPY . .

# Set environment variables

CMD ["bun", "src/index.ts"]
