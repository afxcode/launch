package logger

import (
	"fmt"
	"net/url"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	level zap.AtomicLevel
	Log   *zap.Logger
)

func NewCustomEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:   "msg",
		LevelKey:     "level",
		EncodeLevel:  zapcore.CapitalColorLevelEncoder,
		TimeKey:      "ts",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		CallerKey:    "caller",
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
}

func Init(verbose bool, outpath string) (err error) {
	level = zap.NewAtomicLevel()

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig = NewCustomEncoderConfig()
	config.Level = level
	config.DisableCaller = false

	if verbose {
		level.SetLevel(zapcore.DebugLevel)
	}

	if outpath != "" {
		ll := lumberjack.Logger{
			Filename:   outpath,
			MaxSize:    5, // MB
			MaxBackups: 50,
			MaxAge:     90, // days
			Compress:   true,
		}
		zap.RegisterSink("lumberjack", func(*url.URL) (zap.Sink, error) {
			return lumberjackSink{Logger: &ll}, nil
		})

		config.OutputPaths = []string{fmt.Sprintf("lumberjack:%s", outpath)}
		config.Encoding = "json"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	if runtime.GOOS == "windows" {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	logger, err := config.Build(zap.AddCallerSkip(1))
	zap.ReplaceGlobals(logger)
	Log = logger
	return
}

func SetLevel(l zapcore.Level) {
	level.SetLevel(l)
}

func IsDebug() bool {
	return level.Level() == zapcore.DebugLevel
}

type lumberjackSink struct {
	*lumberjack.Logger
}

func (lumberjackSink) Sync() error {
	return nil
}

// D Debug
func D(msg string, fields ...zap.Field) {
	removeDuplicateKey(fields...)
	Log.Debug(msg, fields...)
}

// I Info
func I(msg string, fields ...zap.Field) {
	removeDuplicateKey(fields...)
	Log.Info(msg, fields...)
}

// W Warning
func W(msg string, fields ...zap.Field) {
	removeDuplicateKey(fields...)
	Log.Warn(msg, fields...)
}

// E Error
func E(msg string, fields ...zap.Field) {
	removeDuplicateKey(fields...)
	Log.Error(msg, fields...)
}

// P Panic
func P(msg string, fields ...zap.Field) {
	removeDuplicateKey(fields...)
	Log.Panic(msg, fields...)
}

// F Field
func F(key string, value any) zap.Field {
	return zap.Any(key, value)
}

func removeDuplicateKey(fields ...zap.Field) {
	cursor := make(map[string]int)
	cursor["msg"] = 0
	for i, field := range fields {
		if _, ok := cursor[field.Key]; !ok {
			cursor[field.Key] = 0
		} else {
			cursor[field.Key]++
		}
		if cursor[field.Key] > 0 {
			fields[i].Key = fmt.Sprintf("%s%d", field.Key, cursor[field.Key])
		}
	}
}
