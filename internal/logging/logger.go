package logging

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	component string
}

type Fields map[string]interface{}

var (
	defaultLogger *Logger
)

type LogLevel string

const (
	// Debug level
	Debug LogLevel = "debug"
	// Info level
	Info LogLevel = "info"
	// Warn level
	Warn LogLevel = "warn"
	// Error level
	Error LogLevel = "error"
	// Fatal level
	Fatal LogLevel = "fatal"
)

type Config struct {
	Level      LogLevel
	Format     string
	Output     io.Writer
	TimeFormat string
}

func DefaultConfig() Config {
	return Config{
		Level:      Info,
		Format:     "json",
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
	}
}

func Init(component string, config Config) {
	logger := logrus.New()

	logger.SetOutput(config.Output)

	switch config.Level {
	case Debug:
		logger.SetLevel(logrus.DebugLevel)
	case Info:
		logger.SetLevel(logrus.InfoLevel)
	case Warn:
		logger.SetLevel(logrus.WarnLevel)
	case Error:
		logger.SetLevel(logrus.ErrorLevel)
	case Fatal:
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimeFormat,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: config.TimeFormat,
			FullTimestamp:   true,
		})
	}

	defaultLogger = &Logger{
		Logger:    logger,
		component: component,
	}
}

func GetLogger() *Logger {
	if defaultLogger == nil {
		Init("app", DefaultConfig())
	}
	return defaultLogger
}

func WithComponent(component string) *Logger {
	if defaultLogger == nil {
		Init("app", DefaultConfig())
	}

	newLogger := *defaultLogger
	newLogger.component = component
	return &newLogger
}

func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	allFields := logrus.Fields{
		"component": l.component,
	}
	_, file, line, ok := runtime.Caller(1)
	if ok {
		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
		allFields["caller"] = fmt.Sprintf("%s:%d", file, line)
	}

	for k, v := range fields {
		allFields[k] = v
	}

	return l.Logger.WithFields(allFields)
}

func (l *Logger) Debug(msg string, fields ...Fields) {
	if len(fields) > 0 {
		l.WithFields(fields[0]).Debug(msg)
	} else {
		l.WithFields(Fields{}).Debug(msg)
	}
}

func (l *Logger) Info(msg string, fields ...Fields) {
	if len(fields) > 0 {
		l.WithFields(fields[0]).Info(msg)
	} else {
		l.WithFields(Fields{}).Info(msg)
	}
}

func (l *Logger) Warn(msg string, fields ...Fields) {
	if len(fields) > 0 {
		l.WithFields(fields[0]).Warn(msg)
	} else {
		l.WithFields(Fields{}).Warn(msg)
	}
}

func (l *Logger) Error(msg string, err error, fields ...Fields) {
	mergedFields := Fields{}
	if len(fields) > 0 {
		mergedFields = fields[0]
	}
	if err != nil {
		mergedFields["error"] = err.Error()
	}
	l.WithFields(mergedFields).Error(msg)
}

func (l *Logger) Fatal(msg string, err error, fields ...Fields) {
	mergedFields := Fields{}
	if len(fields) > 0 {
		mergedFields = fields[0]
	}
	if err != nil {
		mergedFields["error"] = err.Error()
	}
	l.WithFields(mergedFields).Fatal(msg)
}
