# Building Cligram from Source

This guide explains how to compile and run Cligram directly from source code.

## Prerequisites

Before building, ensure you have the following installed on your system:

- **Go:** Version 1.25 or higher ([download Go](https://go.dev/dl/))
- **C/C++ Compiler:** `gcc` or `clang` (required for Cgo audio/voice calling bindings)
- **Make:** Standard build automation tool
- **Git:** Version control

## Build Instructions

1. **Clone the repository:**
   ```bash
   git clone https://github.com/Kumneger0/cligram.git
   cd cligram
   ```

2. **Build the binary:**
   ```bash
   make build
   ```
   *Note:* Pre-compiled static VoIP dependencies are automatically fetched during `make build`.

3. **(Optional) Install system-wide:**
   ```bash
   sudo make install
   ```
   Alternatively, copy the generated `cligram` binary to `~/.local/bin/` or `/usr/local/bin/`.

## Verifying the Build

Verify that Cligram starts properly:

```bash
./cligram version
```

You can test microphone and speaker loopback with:

```bash
./cligram call-helper --loopback
```
