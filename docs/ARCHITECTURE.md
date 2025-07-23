# ScreenTime Widget Architecture

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
│       ├── types/             # TypeScript type definitions
│       │   └── usage.ts       # Usage data types (mirrors Go types)
│       ├── utils/             # Utility functions
│       │   └── timeFormatter.ts      # Time formatting utilities
│       ├── constants/         # Application constants
│       │   └── app.ts         # App configuration constants
│       ├── App.tsx            # Root component
│       ├── main.tsx           # React entry point
│       └── index.css          # Global styles
├── scripts/                   # Build and deployment scripts
├── docs/                      # Documentation
│   └── ARCHITECTURE.md        # This file
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── wails.json                 # Wails configuration
└── README.md                  # Project documentation
```

## Architecture Principles

### Backend (Go)

1. **Layered Architecture**: Clear separation between app, services, and platform layers
2. **Dependency Injection**: Services are injected into the app layer
3. **Interface Segregation**: Platform-specific code is abstracted behind interfaces
4. **Single Responsibility**: Each package has a single, well-defined purpose

### Frontend (React/TypeScript)

1. **Component-Based**: UI is broken down into reusable components
2. **Custom Hooks**: Business logic is extracted into custom hooks
3. **Type Safety**: Full TypeScript coverage with proper type definitions
4. **Separation of Concerns**: Components, hooks, utils, and types are separated

### Key Design Decisions

1. **Platform Abstraction**: Windows API calls are abstracted behind an interface for future cross-platform support
2. **Service Layer**: Screen time tracking logic is encapsulated in a dedicated service
3. **Type Mirroring**: Frontend types mirror backend types for consistency
4. **Error Boundaries**: Proper error handling with React error boundaries
5. **Configuration**: Centralized configuration constants

## Data Flow

1. **Startup**: App initializes and starts the screen time tracker service
2. **Tracking**: Service polls Windows API every second to track active window
3. **Storage**: Usage data is stored in memory with thread-safe access
4. **Frontend Updates**: React component polls backend every 5 seconds for updates
5. **Display**: Widget displays formatted usage data with proper error handling

## Future Extensibility

- **Cross-Platform**: Platform interface allows easy addition of macOS/Linux support
- **Data Persistence**: Service layer can be extended to save/load data
- **Additional Features**: New services can be added without affecting existing code
- **UI Themes**: Component structure supports easy theming
- **Testing**: Clear separation allows for comprehensive unit testing
