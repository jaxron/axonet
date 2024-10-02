package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// Logger interface defines the logging functionality required by the client.
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	WithFields(fields ...Field) Logger
}

// Field creators.
func String(key string, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// NoOpLogger is a logger that does nothing, used as a default when no logger is provided.
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(_ string)                    {}
func (l *NoOpLogger) Info(_ string)                     {}
func (l *NoOpLogger) Warn(_ string)                     {}
func (l *NoOpLogger) Error(_ string)                    {}
func (l *NoOpLogger) Debugf(_ string, _ ...interface{}) {}
func (l *NoOpLogger) Infof(_ string, _ ...interface{})  {}
func (l *NoOpLogger) Warnf(_ string, _ ...interface{})  {}
func (l *NoOpLogger) Errorf(_ string, _ ...interface{}) {}
func (l *NoOpLogger) WithFields(_ ...Field) Logger      { return l }

// BasicLogger uses the standard library log package for logging.
type BasicLogger struct {
	logger *log.Logger
	fields []Field
}

// NewBasicLogger creates a new BasicLogger that writes to stdout.
func NewBasicLogger() Logger {
	return &BasicLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		fields: []Field{},
	}
}

func (l *BasicLogger) log(level, msg string) {
	if len(l.fields) > 0 {
		fieldStrings := make([]string, len(l.fields))
		for i, f := range l.fields {
			fieldStrings[i] = fmt.Sprintf("%s=%v", f.Key, f.Value)
		}
		l.logger.Printf("%s: %s | %s", level, msg, strings.Join(fieldStrings, " "))
	} else {
		l.logger.Printf("%s: %s", level, msg)
	}
}

func (l *BasicLogger) logf(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.log(level, msg)
}

func (l *BasicLogger) Debug(msg string)                          { l.log("DEBUG", msg) }
func (l *BasicLogger) Info(msg string)                           { l.log("INFO", msg) }
func (l *BasicLogger) Warn(msg string)                           { l.log("WARN", msg) }
func (l *BasicLogger) Error(msg string)                          { l.log("ERROR", msg) }
func (l *BasicLogger) Debugf(format string, args ...interface{}) { l.logf("DEBUG", format, args...) }
func (l *BasicLogger) Infof(format string, args ...interface{})  { l.logf("INFO", format, args...) }
func (l *BasicLogger) Warnf(format string, args ...interface{})  { l.logf("WARN", format, args...) }
func (l *BasicLogger) Errorf(format string, args ...interface{}) { l.logf("ERROR", format, args...) }

func (l *BasicLogger) WithFields(fields ...Field) Logger {
	return &BasicLogger{
		logger: l.logger,
		fields: append(l.fields, fields...),
	}
}
