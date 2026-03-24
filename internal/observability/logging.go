package observability

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger provides structured logging
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// Info logs an info message
	Info(msg string, fields ...Field)

	// Warn logs a warning message
	Warn(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, fields ...Field)

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...Field)

	// With creates a child logger with additional fields
	With(fields ...Field) Logger

	// Sync flushes any buffered log entries
	Sync() error
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// StringField creates a string field
func StringField(key string, value string) Field {
	return Field{Key: key, Value: value}
}

// IntField creates an int field
func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// ErrorField creates an error field
func ErrorField(err error) Field {
	return Field{Key: "error", Value: err}
}

// DurationField creates a duration field
func DurationField(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// zapLogger wraps zap.Logger
type zapLogger struct {
	logger *zap.Logger
}

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level      string
	Format     string
	OutputPath string
}

// NewLogger creates a new logger
func NewLogger(cfg LoggerConfig) (Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create output
	var output zapcore.WriteSyncer
	if cfg.OutputPath != "" {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	} else {
		output = zapcore.AddSync(os.Stdout)
	}

	// Create core
	core := zapcore.NewCore(encoder, output, level)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &zapLogger{logger: logger}, nil
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() Logger {
	logger, _ := NewLogger(LoggerConfig{
		Level:  "info",
		Format: "json",
	})
	return logger
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.convertFields(fields)...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.convertFields(fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.convertFields(fields)...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, l.convertFields(fields)...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, l.convertFields(fields)...)
}

func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{logger: l.logger.With(l.convertFields(fields)...)}
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

func (l *zapLogger) convertFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			zapFields = append(zapFields, zap.String(f.Key, v))
		case int:
			zapFields = append(zapFields, zap.Int(f.Key, v))
		case error:
			zapFields = append(zapFields, zap.Error(v))
		case time.Duration:
			zapFields = append(zapFields, zap.Duration(f.Key, v))
		default:
			zapFields = append(zapFields, zap.Any(f.Key, v))
		}
	}
	return zapFields
}

// NoopLogger is a logger that does nothing
type NoopLogger struct{}

func (l *NoopLogger) Debug(msg string, fields ...Field) {}
func (l *NoopLogger) Info(msg string, fields ...Field)  {}
func (l *NoopLogger) Warn(msg string, fields ...Field)  {}
func (l *NoopLogger) Error(msg string, fields ...Field) {}
func (l *NoopLogger) Fatal(msg string, fields ...Field) {}
func (l *NoopLogger) With(fields ...Field) Logger       { return l }
func (l *NoopLogger) Sync() error                       { return nil }
