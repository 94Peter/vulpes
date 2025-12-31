// Package log provides a simplified and opinionated interface for structured logging,
// built on top of the high-performance zap logger.
package log

// --- Structured Logging Methods (using zap.Logger) ---
// These methods are highly performant and are the preferred way to log in production.

// Debug logs a message at the Debug level with structured fields.
func Debug(msg string, fields ...Field) {
	l().Debug(msg, fields...)
}

// Info logs a message at the Info level with structured fields.
func Info(msg string, fields ...Field) {
	l().Info(msg, fields...)
}

// Warn logs a message at the Warn level with structured fields.
func Warn(msg string, fields ...Field) {
	l().Warn(msg, fields...)
}

// Error logs a message at the Error level with structured fields.
func Error(msg string, fields ...Field) {
	l().Error(msg, fields...)
}

// Panic logs a message at the Panic level and then panics.
func Panic(msg string, fields ...Field) {
	l().Panic(msg, fields...)
}

// Fatal logs a message at the Fatal level and then calls os.Exit(1).
func Fatal(msg string, fields ...Field) {
	l().Fatal(msg, fields...)
}

// --- Formatted Logging Methods (using zap.SugaredLogger) ---
// These methods provide a convenient Printf-style API but are slightly less performant.

// Debugf logs a formatted message at the Debug level.
func Debugf(format string, a ...any) {
	s().Debugf(format, a...)
}

// Infof logs a formatted message at the Info level.
func Infof(format string, a ...any) {
	s().Infof(format, a...)
}

// Warnf logs a formatted message at the Warn level.
func Warnf(format string, a ...any) {
	s().Warnf(format, a...)
}

// Errorf logs a formatted message at the Error level.
func Errorf(format string, a ...any) {
	s().Errorf(format, a...)
}

// Fatalf logs a formatted message at the Fatal level and then calls os.Exit(1).
func Fatalf(format string, a ...any) {
	s().Fatalf(format, a...)
}
