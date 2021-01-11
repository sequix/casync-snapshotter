package log

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	flagNoStdout   = flag.Bool("log-no-stdout", false, "do not output log to stdout")
	flagJSON       = flag.Bool("log-json", false, "log in json format")
	flagTimestamp  = flag.Bool("log-timestamp", false, "log milliseconds timestamp instead of ISO8601")
	flagLevel      = flag.String("log-level", zap.InfoLevel.String(), "log level")
	flagDir        = flag.String("log-dir", "./logs", "which directory log files go to")
	flagNoCompress = flag.Bool("log-no-compress", false, "do not compress rotated logs")
	flagRotateSize = flag.Int("log-rotate-size", 100, "maximum size in megabytes of the log file before it gets rotated, 0 for not rotating")
	flagRetainDays = flag.Int("log-retain-days", 180, "maximum number of days that logs will be kept, based on the date encoded in their names, 0 for kept forever")
	flagMaxRotates = flag.Int("log-max-rotates", 100, "maximum number of log files that will be kept, 0 for unlimited number")
)

type Logger zap.SugaredLogger

var G *Logger

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("init dev logger: %s", err))
	}
	G = (*Logger)(logger.Sugar())
}

func Init(module string) {
	switch *flagLevel {
	case "info", "warn", "error", "fatal", "debug":
	default:
		panic(fmt.Sprintf(`log level %s invlaid, expected one of "info", "warn", "error", "fatal" and "debug"`, *flagLevel))
	}

	lv := zap.NewAtomicLevel()
	if err := lv.UnmarshalText([]byte(*flagLevel)); err != nil {
		panic(fmt.Sprintf(`log level %s invlaid: %s`, *flagLevel, err))
	}

	if err := os.MkdirAll(*flagDir, 0775); err != nil {
		panic(fmt.Sprintf("mkdir %s for logs: %s", *flagDir, err))
	}

	config := zap.NewProductionEncoderConfig()
	if *flagTimestamp {
		config.EncodeTime = func(t time.Time, p zapcore.PrimitiveArrayEncoder) {
			p.AppendInt64(t.UnixNano() / int64(time.Millisecond))
		}
	} else {
		config.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	var encoder zapcore.Encoder
	if *flagJSON {
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		encoder = zapcore.NewConsoleEncoder(config)
	}

	var fileWriter zapcore.WriteSyncer
	logFilename := filepath.Join(*flagDir, module+".log")

	fi, err := os.Stat(logFilename)
	if err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("stat %s: %s", logFilename, err))
	}
	if err == nil && fi.IsDir() {
		panic(fmt.Sprintf("expected %s to be a file or not exist", logFilename))
	}

	if *flagRotateSize > 0 {
		fileWriter = zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFilename,
			MaxSize:    *flagRotateSize,
			MaxAge:     *flagRetainDays,
			MaxBackups: *flagMaxRotates,
			Compress:   !*flagNoCompress,
		})
	} else {
		logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
		if err != nil {
			panic(fmt.Sprintf("open log file %s: %s", logFilename, err))
		}
		fileWriter = zapcore.AddSync(logFile)
	}
	core := zapcore.NewCore(encoder, fileWriter, lv)

	if !*flagNoStdout {
		core = zapcore.NewTee(core, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lv))
	}
	G = (*Logger)(zap.New(core, zap.AddStacktrace(zap.ErrorLevel)).Named(module).Sugar())
}

type contextKeyType interface{}

var contextKey = contextKeyType(struct{}{})

func C(ctx context.Context) *Logger {
	return ctx.Value(contextKey).(*Logger)
}

func (l *Logger) Inject(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey, l)
}

func (l *Logger) toZap() *zap.SugaredLogger {
	return (*zap.SugaredLogger)(l)
}

func (l *Logger) AddCallerSkip(n int) *Logger {
	return (*Logger)(l.toZap().Desugar().WithOptions(zap.AddCallerSkip(n)).Sugar())
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
