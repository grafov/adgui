# adgui - AdGuard VPN GUI Client

## Project Overview

adgui is a simple GUI client for controlling the AdGuard VPN CLI tool on Linux systems. It provides a system tray interface that allows users to connect to VPN locations, disconnect, and view connection status without needing to use the command line.

### Key Features
- System tray icon showing VPN connection status (connected/disconnected)
- Connect to VPN locations with "Connect Auto" (chooses fastest location based on ping)
- Connect to specific locations from a list
- Disconnect from VPN
- Location selector window with ping times
- Status checking every 10 seconds

### Technology Stack
- **Language**: Go (Golang)
- **GUI Framework**: Fyne.io/v2 (cross-platform GUI toolkit)
- **Platform**: Linux (X11/Wayland)
- **Dependencies**: 
  - fyne.io/fyne/v2 - Main GUI framework
  - fyne.io/systray - System tray functionality
  - Various Go standard library packages

### Architecture
The application consists of:
- Main application entry point: `cmd/adgui/main.go`
- Theme assets: PNG icons in `theme/` directory
- Build system: Makefile for compilation and management

## Building and Running

### Prerequisites
- Go 1.24 or higher
- AdGuard VPN CLI (`adguardvpn-cli`) installed and available in PATH
- Linux system with X11/Wayland support

### Build Commands
```bash
# Build the application
make build

# Run the application (builds first)
make run

# Run with debug logging
make run-log

# Clean build artifacts
make clean

# Run tests
make test

# Run race condition detection
make run-race

# Lint the code
make lint

# Tidy Go modules
make tidy

# Install to /usr/local/bin
make install
```

### Environment Variables
- `ADGUARD_CMD` - Path to the adguardvpn-cli binary (defaults to "adguardvpn-cli")
- `SLOG_LEVEL` - Logging level (set to "debug" when using run-log target)

## Development Conventions

### Code Structure
- Main application logic is in `cmd/adgui/main.go`
- Theme and icon assets are handled in `theme/theme.go`
- The VPNManager struct handles all VPN interactions and UI updates
- System tray functionality is implemented using Fyne desktop app interface

### Important Implementation Notes
- The application uses goroutines for background status checking (every 10 seconds)
- UI updates happen through a channel-based system to ensure thread safety
- Location parsing handles ANSI escape codes from the CLI output
- Icons change based on connection status (theme/icon-on.png for connected, theme/icon-off.png for disconnected)

### Testing
- Tests are run with Ginkgo framework (`make test`)
- The application has race condition detection capability (`make run-race`)

### Debugging
- Debug logging is available with `SLOG_LEVEL=debug`
- The application logs status changes and command execution results to stdout
- Status checker includes debug prints to show connection state changes

## Project Status
The README mentions: "Works partially yet. Development just in progress." This indicates the project is in active development and may have some incomplete features.

## Files and Directories
- `go.mod/go.sum` - Go module definition and dependency versions
- `Makefile` - Build and management commands
- `cmd/adgui/main.go` - Main application source code
- `theme/` - Theme assets and icon handling
- `res/` - Resource files (SVG icons)
- `README.md` - Basic project description
- `.gitignore` - Git ignore patterns
- `adgui.code-workspace` - VS Code workspace configuration
- `adgui-run` - Convenience script for running a single instance