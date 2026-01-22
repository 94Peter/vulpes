// Package log provides a simplified and opinionated interface for structured logging,
// built on top of the high-performance zap logger.
// It offers a global logger instance that can be configured once and used throughout the application.
package log

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// baseLogger is the underlying zap.Logger instance.
	baseLogger *zap.Logger
	// sugar is the zap.SugaredLogger, providing a more ergonomic, but slightly slower, API.
	sugar *zap.SugaredLogger
	// once ensures that the logger initialization occurs only once.
	once sync.Once
)

// Config holds the configuration for the logger.
type Config struct {
	Level       string
	ServiceName string
	Env         string
	CallerSkip  int
	Development bool
}

const defaultCallerSkip = 2

// defaultConfig provides a sensible default configuration for the logger.
var defaultConfig = Config{
	Development: true,
	Level:       "debug",
	CallerSkip:  defaultCallerSkip,
	Env:         "dev",
}

// SetConfig applies user-defined options to the default logger configuration.
// This should be called before the first log message is written to have an effect.
func SetConfig(opts ...Option) {
	for _, opt := range opts {
		opt(&defaultConfig)
	}
}

// _Init initializes the global logger instance based on the provided configuration.
// It uses sync.Once to ensure thread-safety and that initialization happens only once.
func _Init(cfg Config) {
	once.Do(func() {
		var level zapcore.Level
		switch cfg.Level {
		case "debug":
			level = zapcore.DebugLevel
		case "warn":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		default:
			level = zapcore.InfoLevel
		}

		var zapCfg zap.Config
		if cfg.Development {
			// Development config: human-friendly, colored output.
			zapCfg = zap.NewDevelopmentConfig()
			zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			zapCfg.EncoderConfig.EncodeTime = timeFormatter
		} else {
			// Production config: structured JSON, optimized for performance.
			zapCfg = zap.NewProductionConfig()
			zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			zapCfg.EncoderConfig.TimeKey = "ts"
		}
		zapCfg.Level = zap.NewAtomicLevelAt(level)
		var err error
		baseLogger, err = zapCfg.Build()
		if err != nil {
			panic(err)
		}
		// AddCallerSkip(2) is used to make the caller information point to the actual call site
		// (e.g., log.Info) rather than the wrapper function inside this package.
		baseLogger = baseLogger.
			WithOptions(zap.AddCallerSkip(cfg.CallerSkip))
		if cfg.ServiceName != "" {
			baseLogger = baseLogger.With(zap.String("service", cfg.ServiceName))
			baseLogger = baseLogger.With(zap.String("env", cfg.Env))
		}

		sugar = baseLogger.Sugar()
	})
}

// l returns the singleton instance of the high-performance, structured zap.Logger.
// If the logger is not already initialized, it will be initialized with the default configuration.
func l() *zap.Logger {
	if baseLogger == nil {
		_Init(defaultConfig)
	}
	return baseLogger
}

// s returns the singleton instance of the ergonomic zap.SugaredLogger.
// It is convenient for formatted logging (e.g., Infof) but is slightly less performant.
func s() *zap.SugaredLogger {
	if sugar == nil {
		_Init(defaultConfig)
	}
	return sugar
}

// timeFormatter provides a custom time format for development logs.
func timeFormatter(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
