package logger

import (
	"context"
	"fmt"

	"go-starter/internal/constants"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Field is a structured log field. Build them with the New* helpers below so
// call sites never import zap directly.
type Field = zap.Field

// Logger provides level based, structured logging.
//
// Every method takes a `bucket` (the component, e.g. "SQS", "UserRepository")
// and a `handler` (the function name). That pair makes logs greppable and is
// the same convention used across the service.
type Logger interface {
	Info(ctx context.Context, bucket, handler, message string, additionalFields ...Field)
	Warn(ctx context.Context, bucket, handler, message string, additionalFields ...Field)
	Error(ctx context.Context, bucket, handler, message string, additionalFields ...Field)
	Debug(ctx context.Context, bucket, handler, message string, additionalFields ...Field)
}

type zapLogger struct {
	zap *zap.Logger
}

// New builds a JSON structured logger writing to stdout.
func New(level string) Logger {
	lvl := zap.InfoLevel
	switch level {
	case "debug":
		lvl = zap.DebugLevel
	case "warn":
		lvl = zap.WarnLevel
	case "error":
		lvl = zap.ErrorLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	z, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Sprintf("failed to build logger: %v", err))
	}
	return &zapLogger{zap: z}
}

func (l *zapLogger) Info(ctx context.Context, bucket, handler, message string, fields ...Field) {
	l.zap.Info(message, l.fields(ctx, bucket, handler, fields)...)
}

func (l *zapLogger) Warn(ctx context.Context, bucket, handler, message string, fields ...Field) {
	l.zap.Warn(message, l.fields(ctx, bucket, handler, fields)...)
}

func (l *zapLogger) Error(ctx context.Context, bucket, handler, message string, fields ...Field) {
	l.zap.Error(message, l.fields(ctx, bucket, handler, fields)...)
}

func (l *zapLogger) Debug(ctx context.Context, bucket, handler, message string, fields ...Field) {
	l.zap.Debug(message, l.fields(ctx, bucket, handler, fields)...)
}

// fields prefixes every log line with bucket/handler and, when present on the
// context, the trace and request IDs.
func (l *zapLogger) fields(ctx context.Context, bucket, handler string, extra []Field) []Field {
	out := make([]Field, 0, len(extra)+4)
	out = append(out, NewString("bucket", bucket), NewString("handler", handler))

	if ctx != nil {
		if v := ctx.Value(constants.TraceID); v != nil {
			out = append(out, NewInterface("trace_id", v))
		}
		if v := ctx.Value(constants.RequestID); v != nil {
			out = append(out, NewInterface("request_id", v))
		}
	}
	return append(out, extra...)
}

// ── Field constructors ───────────────────────────────────────────────────────

func NewString(key, value string) Field           { return zap.String(key, value) }
func NewInt(key string, value int) Field          { return zap.Int(key, value) }
func NewInt64(key string, value int64) Field      { return zap.Int64(key, value) }
func NewFloat64(key string, value float64) Field  { return zap.Float64(key, value) }
func NewBool(key string, value bool) Field        { return zap.Bool(key, value) }
func NewInterface(key string, value any) Field    { return zap.Any(key, value) }
func NewError(err error) Field                    { return zap.Error(err) }
func NewStrings(key string, value []string) Field { return zap.Strings(key, value) }
