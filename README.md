# ScreenTime Widget for Windows

A clean, minimalist desktop widget that tracks and displays your daily screen time and most-used applications, built with Wails, Go, React, and TypeScript.

## Features

- **Real-time Tracking**: Monitors active applications every second
- **Clean UI**: Frameless, translucent widget that stays on top
- **Daily Statistics**: Shows total screen time and top 4 most-used apps
- **Lightweight**: Minimal CPU and memory footprint
- **Draggable**: Move the widget anywhere on your desktop
- **Auto-refresh**: Updates every 5 seconds

## Project Structure

```
qwin/
├── internal/                    # Go backend (private packages)
│   ├── app/                    # Application layer
│   │   └── app.go             # Main app struct and lifecycle
│   ├── services/              # Business logic layer
│   │   └── screentime_tracker.go  # Screen time tracking service
│   ├── platform/              # Platform-specific implementations
│   │   ├── interface.go       # Platform interface definitions
│   │   └── windows_api.go     # Windows API implementation
│   └── types/                 # Shared type definitions
│       └── usage.go           # Usage data types
├── frontend/                   # React frontend
│   └── src/
│       ├── components/        # React components
│       │   ├── ScreenTimeWidget.tsx  # Main widget component
│       │   └── ErrorBoundary.tsx     # Error handling component
│       ├── hooks/             # Custom React hooks
│       │   └── useScreenTime.ts      # Screen time data hook
│       ├── utils/             # Utility functions
│       │   └── timeFormatter.ts      # Time formatting utilities
│       ├── constants/         # Application constants
│       │   └── app.ts         # App configuration constants
│       ├── App.tsx            # Root component
│       ├── main.tsx           # React entry point
│       └── index.css          # Global styles
├── docs/                      # Documentation
│   └── ARCHITECTURE.md        # Architecture documentation
├── scripts/                   # Build and deployment scripts
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── wails.json                 # Wails configuration
└── README.md                  # This file
```

## Architecture Principles

### Backend (Go)

- **Layered Architecture**: Clear separation between app, services, and platform layers
- **Dependency Injection**: Services are injected into the app layer
- **Interface Segregation**: Platform-specific code is abstracted behind interfaces
- **Single Responsibility**: Each package has a single, well-defined purpose

### Frontend (React/TypeScript)

- **Component-Based**: UI is broken down into reusable components
- **Custom Hooks**: Business logic is extracted into custom hooks
- **Type Safety**: Full TypeScript coverage with proper type definitions
- **Separation of Concerns**: Components, hooks, utils, and constants are separated

## Building

### Prerequisites

- Go 1.21 or later
- Node.js 16 or later
- Wails CLI v2

### Development

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Run in development mode
wails dev
```

### Production Build

```bash
# Build for Windows
wails build

# Build with clean
wails build -clean
```

## Usage

1. Run the executable (`qwin.exe`)
2. The widget will appear as a small, translucent window
3. Drag it to your preferred position on the desktop
4. The widget will automatically track your application usage
5. Click the minimize button to minimize to system tray
6. Click the X button to close the application

## Technical Details

- **Windows API Integration**: Uses Windows API calls to track active windows
- **Thread-Safe**: Concurrent access to usage data is properly synchronized
- **Memory Efficient**: In-memory storage with minimal overhead
- **Real-time Updates**: Frontend polls backend every 5 seconds for updates
- **Error Handling**: Comprehensive error boundaries and error states

## Future Enhancements

- Data persistence across application restarts
- Weekly/monthly usage reports
- Application time limits and notifications
- Cross-platform support (macOS, Linux)
- System tray integration
- Usage export functionality

## License

MIT License - see LICENSE file for details
