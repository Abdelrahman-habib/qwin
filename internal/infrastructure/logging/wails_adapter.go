package logging

// WailsLoggerAdapter adapts our structured logger to implement the Wails Logger interface
type WailsLoggerAdapter struct {
	logger Logger
}

// NewWailsLoggerAdapter creates a new Wails logger adapter using our structured logger
func NewWailsLoggerAdapter(logger Logger) *WailsLoggerAdapter {
	if logger == nil {
		logger = NewDefaultLogger()
	}
	return &WailsLoggerAdapter{
		logger: logger,
	}
}

// Print logs a message at INFO level (Wails general output)
func (w *WailsLoggerAdapter) Print(message string) {
	w.logger.Info(message, "source", "wails")
}

// Trace logs a message at DEBUG level (Wails trace output)
func (w *WailsLoggerAdapter) Trace(message string) {
	w.logger.Debug(message, "source", "wails", "level", "trace")
}

// Debug logs a message at DEBUG level
func (w *WailsLoggerAdapter) Debug(message string) {
	w.logger.Debug(message, "source", "wails")
}

// Info logs a message at INFO level
func (w *WailsLoggerAdapter) Info(message string) {
	w.logger.Info(message, "source", "wails")
}

// Warning logs a message at WARN level
func (w *WailsLoggerAdapter) Warning(message string) {
	w.logger.Warn(message, "source", "wails")
}

// Error logs a message at ERROR level
func (w *WailsLoggerAdapter) Error(message string) {
	w.logger.Error(message, "source", "wails")
}

// Fatal logs a message at ERROR level (we don't want Wails to actually exit)
func (w *WailsLoggerAdapter) Fatal(message string) {
	w.logger.Error(message, "source", "wails", "level", "fatal")
}
