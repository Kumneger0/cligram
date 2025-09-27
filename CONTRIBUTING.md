# Contributing to Cligram CLI

Thank you for considering contributing to Cligram CLI! This project is a pure Go application that provides a terminal-based Telegram client using the Bubbletea framework for the UI and the gotd/td library for Telegram API interactions.

## Project Architecture

1. **UI Layer (Go + Bubbletea)**:

   - Implemented with Go and Bubbletea for the terminal UI
   - Located in `internal/ui/`
   - Handles all terminal interface rendering and user interactions

2. **Telegram Integration**:

   - Telegram API interactions using gotd/td library
   - Located in `internal/telegram/`
   - Handles all Telegram API calls, message processing, and client management

3. **Core Components**:
   - Configuration management in `internal/config/`
   - Logging utilities in `internal/logger/`
   - Notification system in `internal/notification/`

## Prerequisites

1. **Go Environment**:

   - Install Go 1.25 or later

2. **API Credentials**:
   - Visit [Telegram's API page](https://my.telegram.org/apps) to obtain your API ID and API Hash
   - Set these credentials in your environment variables or configuration file

## Getting Started

1. **Fork the Repository**:

   - Fork this repository to your GitHub account

2. **Clone the Repository**:

   ```bash
   git clone https://github.com/YOUR-USERNAME/cligram.git
   cd cligram
   ```

3. **Install Dependencies**:
   ```bash
   # Go dependencies
   go mod download
   ```

## Development Workflow

1. **UI Layer Development (Go + Bubbletea)**:

   - UI components are in `internal/ui/`

2. **Telegram Integration Development**:

   - Telegram API logic is in `internal/telegram/`
   - Use the gotd/td library for Telegram API interactions
   - Follow the existing patterns for client management and message handling

3. **Core Components Development**:
   - Configuration management in `internal/config/`
   - Logging utilities in `internal/logger/`
   - Notification system in `internal/notification/`

## Making Changes

1. **Branching**:

   - Create a new branch for your feature/bugfix

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Development**:

   - Make your changes following the project's coding standards
   - Update documentation if needed
   - Ensure proper error handling and logging

3. **Committing**:

   - Use descriptive commit messages
   - Follow conventional commits format
   - Run `make lint` before committing

4. **Pull Request**:
   - Open a PR against the main repository
   - Include a clear description of changes
   - Reference any related issues
   - Add screenshots if UI changes are involved
   - Ensure all CI checks pass

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [gotd/td Documentation](https://github.com/gotd/td)
- [Telegram API Documentation](https://core.telegram.org/api)

We appreciate your contributions and look forward to your pull requests!
