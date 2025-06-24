# Building from Source

## Prerequisites

Before building cligram from source, ensure you have the following prerequisites installed:

- [Go](https://go.dev/dl/) version 1.24 or higher
- [Bun](https://bun.sh/) version 1.2 or higher
- Git

## Installation Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/kumneger/cligram.git
   cd cligram
   ```

2. Build the project :
   ```bash
   make build
   ```

3. (Optional) Install the binary system-wide:
   ```bash
   sudo make install
   ```

## Verifying Installation

After installation, you can verify that cligram is working correctly by running:
```bash
cligram --version
```

If you encounter any issues during installation:
For additional help, please [open an issue](https://github.com/kumneger/cligram/issues/new) on GitHub.
