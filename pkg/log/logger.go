package log

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger zap.SugaredLogger

var (
	logJSON       = flag.Bool("log-json", false, "log in json format")
	logTimestamp  = flag.Bool("log-timestamp", false, "log milliseconds timestamp instead of ISO8601")
	logLevel      = flag.String("log-level", zap.InfoLevel.String(), "log level")
	logDir        = flag.String("log-dir", "./logs", "which directory log files go to")
	logRotate     = flag.Bool("log-rotate", false, "rotate log files")
	logRotateSize = flag.Int("log-rotate-size", 100, "maximum size in megabytes of the log file before it gets rotated")
)

var (
	globalLoggerZap *zap.SugaredLogger
	globalLogger    *Logger
)

func Init(module string) {
	switch *logLevel {
	case "info", "warn", "error", "fatal", "debug":
	default:
		panic(fmt.Sprintf(`log level %s invlaid, expected one of "info", "warn", "error", "fatal" and "debug"`, *logLevel))
	}

	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
		panic(fmt.Sprintf(`log level %s invlaid: %s`, *logLevel, err))
	}

	if err := os.MkdirAll(*logDir, 0775); err != nil {
		panic(fmt.Sprintf("mkdir %s for logs: %s", *logDir, err))
	}

	config := zap.NewProductionEncoderConfig()
	if *logTimestamp {
		config.EncodeTime = func(t time.Time, p zapcore.PrimitiveArrayEncoder) {
			p.AppendInt64(t.UnixNano() / int64(time.Millisecond))
		}
	} else {
		config.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	var encoder zapcore.Encoder
	if *logJSON {
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		encoder = zapcore.NewConsoleEncoder(config)
	}

	var fileWriter zapcore.WriteSyncer
	logFilename := filepath.Join(*logDir, "dequash.log")

	if *logRotate {
		rotater := &lumberjack.Logger{
			Filename: logFilename,
			MaxSize:  *logRotateSize,
		}
		fileWriter = zapcore.AddSync(rotater)
	} else {
		logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
		if err != nil {
			panic(fmt.Sprintf("open log file %s: %s", logFilename, err))
		}
		fileWriter = zapcore.AddSync(logFile)
	}
	consoleWriter := zapcore.AddSync(os.Stdout)

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, consoleWriter, level),
		zapcore.NewCore(encoder, fileWriter, level),
	)
	globalLoggerZap = zap.New(core, zap.AddStacktrace(zap.ErrorLevel)).Named(module).Sugar()
	globalLogger = (*Logger)(globalLoggerZap)
}

type contextKeyType interface{}

var (
	ContextKey = contextKeyType(struct{}{})
	ContextID  = "ctxid"
)

func C(ctx context.Context) *Logger {
	return ctx.Value(ContextKey).(*Logger)
}

func G(ctx context.Context) *Logger {
	logger, ok := ctx.Value(ContextKey).(*Logger)
	if !ok || logger == nil {
		return G(InjectIntoContext(ctx))
	}
	return logger
}

func InjectIntoContext(ctx context.Context) context.Context {
	// TODO will snapshotter use this ctx in parallel ?
	if logger := ctx.Value(ContextKey); logger != nil {
		return ctx
	}
	logger := globalLogger.With(ContextID, uuid.New().String())
	return context.WithValue(ctx, ContextKey, logger)
}

func GlobalLogger() *Logger {
	return globalLogger
}

func Named(name string) *Logger {
	return (*Logger)(globalLoggerZap.Named(name))
}

func WithError(err error) *Logger {
	return (*Logger)(globalLoggerZap.With("error", err))
}

func With(args ...interface{}) *Logger {
	return (*Logger)(globalLoggerZap.With(args...))
}

func Info(args ...interface{}) {
	globalLoggerZap.Info(args...)
}

func Warn(args ...interface{}) {
	globalLoggerZap.Warn(args...)
}

func Error(args ...interface{}) {
	globalLoggerZap.Error(args...)
}

func Fatal(args ...interface{}) {
	globalLoggerZap.Fatal(args...)
}

func Debug(args ...interface{}) {
	globalLoggerZap.Debug(args...)
}

func Infof(template string, args ...interface{}) {
	globalLoggerZap.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	globalLoggerZap.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	globalLoggerZap.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	globalLoggerZap.Fatalf(template, args...)
}

func Debugf(template string, args ...interface{}) {
	globalLoggerZap.Debugf(template, args...)
}

func (l *Logger) toZap() *zap.SugaredLogger {
	return (*zap.SugaredLogger)(l)
}

func (l *Logger) Named(name string) *Logger {
	return (*Logger)(l.toZap().Named(name))
}

func (l *Logger) WithError(err error) *Logger {
	return (*Logger)(l.toZap().With("error", err))
}

func (l *Logger) With(args ...interface{}) *Logger {
	return (*Logger)(l.toZap().With(args...))
}

func (l *Logger) Info(args ...interface{}) {
	l.toZap().Info(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.toZap().Warn(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.toZap().Error(args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.toZap().Fatal(args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.toZap().Debug(args...)
}

func (l *Logger) Infof(template string, args ...interface{}) {
	l.toZap().Infof(template, args...)
}

func (l *Logger) Warnf(template string, args ...interface{}) {
	l.toZap().Warnf(template, args...)
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	l.toZap().Errorf(template, args...)
}

func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.toZap().Fatalf(template, args...)
}

func (l *Logger) Debugf(template string, args ...interface{}) {
	l.toZap().Debugf(template, args...)
}
