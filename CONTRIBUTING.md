# Contributing to Cligram CLI

Thank you for considering contributing to Cligram CLI! This project uses a unique architecture where Go with Bubbletea handles the UI layer, while TypeScript manages the core functionality, with JSON-RPC over stdio for communication.

## Project Architecture

1. **UI Layer (Go + Bubbletea)**:
   - Implemented with Go and Bubbletea for the terminal UI
   - Located in `internal/ui/`
   - Handles all terminal interface rendering and user interactions

2. **Core Functionality (TypeScript)**:
   - Core Telegram API interactions and business logic
   - Located in `js/src/`
   - Handles all Telegram API calls and data processing

3. **Communication Layer**:
   - JSON-RPC over stdio for UI-Core communication
   - Located in `internal/rpc/`
   - Handles all inter-process communication between UI and Core

## Prerequisites

1. **Go Environment**:
   - Install Go 1.20 or later
   - Set up your GOPATH and GOROOT
   - Install Go tools using `go install` commands in [tools/tools.go](cci:7://file:///home/kumneger/projects/sideProjects/tg-cli/tools/tools.go:0:0-0:0)

2. **Node.js Environment**:
   - Install Node.js 18 or later
   - Install Bun.js for TypeScript development

3. **API Credentials**:
   - Visit [Telegram's API page](https://my.telegram.org/apps) to obtain your API ID and API Hash
   - Set these credentials in your environment variables

## Getting Started

1. **Fork the Repository**:
   - Fork this repository to your GitHub account

2. **Clone the Repository**:
   ```bash
   git clone https://github.com/YOUR-USERNAME/tg-cli.git
   cd tg-cli
   ```

3. **Install Dependencies**:
   ```bash
   # Go dependencies
   go mod download
   
   # TypeScript dependencies
   bun install
   ```

## Development Workflow

1. **UI Layer Development (Go + Bubbletea)**:
   - UI components are in `internal/ui/`
   - Use Bubbletea patterns for UI development
   - Follow Go best practices for terminal applications
   - Test UI interactions thoroughly

2. **Core Layer Development (TypeScript)**:
   - Core logic is in `js/src/`
   - Follow TypeScript best practices
   - Write comprehensive tests for API interactions
   - Ensure proper error handling for Telegram API calls

3. **Inter-process Communication**:
   - JSON-RPC methods are defined in `internal/rpc/`
   - Follow JSON-RPC 2.0 specification
   - Ensure proper error handling in RPC calls
   - Maintain consistent request/response patterns

## Code Style and Standards

1. **Go Code (UI Layer)**:
   - Use `gofmt` for formatting
   - Follow Go naming conventions
   - Write clear and concise docstrings
   - Use error handling consistently
   - Follow Bubbletea patterns for UI components

2. **TypeScript Code (Core Layer)**:
   - Use strict type checking
   - Follow React/TypeScript best practices
   - Write JSDoc comments
   - Use ESLint for linting
   - Maintain proper TypeScript interfaces for RPC communication

## Making Changes

1. **Branching**:
   - Create a new branch for your feature/bugfix
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Development**:
   - Make your changes following the project's coding standards
   - Write tests for new functionality
   - Update documentation if needed
   - Ensure proper RPC method definitions if adding new features

3. **Testing**:
   - Run all tests before submitting
   - Test UI interactions thoroughly
   - Test RPC communication flow
   - Test error handling scenarios

4. **Committing**:
   - Use descriptive commit messages
   - Follow conventional commits format
   - Run `golangci-lint run` and `bun run lint` before committing

5. **Pull Request**:
   - Open a PR against the main repository
   - Include a clear description of changes
   - Reference any related issues
   - Add screenshots if UI changes are involved
   - Document any new RPC methods if added

## Code Review

1. **Review Process**:
   - All PRs require at least one approval
   - Changes will be reviewed for:
     - Code quality and style
     - Test coverage
     - Documentation
     - Security implications
     - Proper RPC method implementation

2. **Feedback**:
   - Be open to feedback and suggestions
   - Address review comments promptly
   - Engage in constructive discussion

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [TypeScript Documentation](https://www.typescriptlang.org/docs/)
- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [Telegram API Documentation](https://core.telegram.org/api)

We appreciate your contributions and look forward to your pull requests!
