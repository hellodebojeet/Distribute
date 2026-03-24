// Package logging provides structured JSON logging for the distributed filesystem.
package logging

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level represents a logging level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger wraps zap logger with additional context.
type Logger struct {
	zap     *zap.Logger
	context []zap.Field
}

// NewLogger creates a new structured logger.
func NewLogger(level Level, development bool) (*Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case LevelDebug:
		zapLevel = zapcore.DebugLevel
	case LevelInfo:
		zapLevel = zapcore.InfoLevel
	case LevelWarn:
		zapLevel = zapcore.WarnLevel
	case LevelError:
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      development,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Add timestamp
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		zap: logger,
	}, nil
}

// NewDevelopmentLogger creates a logger for development with human-readable output.
func NewDevelopmentLogger() (*Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		zap: logger,
	}, nil
}

// With creates a child logger with additional context.
func (l *Logger) With(fields ...Field) *Logger {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = f.toZap()
	}
	return &Logger{
		zap:     l.zap.With(zapFields...),
		context: append(l.context, zapFields...),
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, fieldsToZap(fields)...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, fieldsToZap(fields)...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, fieldsToZap(fields)...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, fieldsToZap(fields)...)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, fieldsToZap(fields)...)
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Field represents a structured log field.
type Field struct {
	key string
	val interface{}
	typ fieldType
}

type fieldType int

const (
	fieldTypeString fieldType = iota
	fieldTypeInt
	fieldTypeInt64
	fieldTypeUint64
	fieldTypeFloat64
	fieldTypeBool
	fieldTypeError
	fieldTypeAny
)

func (f Field) toZap() zap.Field {
	switch f.typ {
	case fieldTypeString:
		return zap.String(f.key, f.val.(string))
	case fieldTypeInt:
		return zap.Int(f.key, f.val.(int))
	case fieldTypeInt64:
		return zap.Int64(f.key, f.val.(int64))
	case fieldTypeUint64:
		return zap.Uint64(f.key, f.val.(uint64))
	case fieldTypeFloat64:
		return zap.Float64(f.key, f.val.(float64))
	case fieldTypeBool:
		return zap.Bool(f.key, f.val.(bool))
	case fieldTypeError:
		return zap.Error(f.val.(error))
	case fieldTypeAny:
		return zap.Any(f.key, f.val)
	default:
		return zap.String(f.key, "unknown")
	}
}

func fieldsToZap(fields []Field) []zap.Field {
	result := make([]zap.Field, len(fields))
	for i, f := range fields {
		result[i] = f.toZap()
	}
	return result
}

// String creates a string field.
func String(key, val string) Field {
	return Field{key: key, val: val, typ: fieldTypeString}
}

// Int creates an int field.
func Int(key string, val int) Field {
	return Field{key: key, val: val, typ: fieldTypeInt}
}

// Int64 creates an int64 field.
func Int64(key string, val int64) Field {
	return Field{key: key, val: val, typ: fieldTypeInt64}
}

// Uint64 creates a uint64 field.
func Uint64(key string, val uint64) Field {
	return Field{key: key, val: val, typ: fieldTypeUint64}
}

// Float64 creates a float64 field.
func Float64(key string, val float64) Field {
	return Field{key: key, val: val, typ: fieldTypeFloat64}
}

// Bool creates a bool field.
func Bool(key string, val bool) Field {
	return Field{key: key, val: val, typ: fieldTypeBool}
}

// Err creates an error field.
func Err(err error) Field {
	return Field{key: "error", val: err, typ: fieldTypeError}
}

// Any creates an any field.
func Any(key string, val interface{}) Field {
	return Field{key: key, val: val, typ: fieldTypeAny}
}

// _CID creates a CID field.
func CID(cid string) Field {
	return String("cid", cid)
}

// Peer creates a peer ID field.
func Peer(peerID string) Field {
	return String("peer", peerID)
}

// Duration creates a duration field.
func Duration(key string, d time.Duration) Field {
	return Int64(key, d.Milliseconds())
}

// Global logger instance.
var global *Logger

func init() {
	var err error
	global, err = NewLogger(LevelInfo, false)
	if err != nil {
		// Fallback to stderr
		panic(err)
	}
}

// SetGlobal sets the global logger.
func SetGlobal(logger *Logger) {
	global = logger
}

// Debug logs a debug message using the global logger.
func Debug(msg string, fields ...Field) {
	global.Debug(msg, fields...)
}

// Info logs an info message using the global logger.
func Info(msg string, fields ...Field) {
	global.Info(msg, fields...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, fields ...Field) {
	global.Warn(msg, fields...)
}

// Error logs an error message using the global logger.
func Error(msg string, fields ...Field) {
	global.Error(msg, fields...)
}

// Fatal logs a fatal message and exits using the global logger.
func Fatal(msg string, fields ...Field) {
	global.Fatal(msg, fields...)
}

// WithContext creates a logger from context.
func WithContext(ctx context.Context) *Logger {
	// Extract trace ID from context if available
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return global.With(String("trace_id", traceID))
	}
	return global
}

// Stdout returns os.Stdout.
func Stdout() *os.File {
	return os.Stdout
}
