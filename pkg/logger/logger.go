package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Interface -.
type Interface interface {
	Debug(message interface{}, args ...interface{})
	Info(message interface{}, args ...interface{})
	Warn(message interface{}, args ...interface{})
	Error(message interface{}, args ...interface{})
	Fatal(message interface{}, args ...interface{})
}

// Logger -.
type Logger struct {
	logger *zerolog.Logger
}

var _ Interface = (*Logger)(nil)

// New -.
func New(level string) *Logger {
	var l zerolog.Level

	switch strings.ToLower(level) {
	case "error":
		l = zerolog.ErrorLevel
	case "warn":
		l = zerolog.WarnLevel
	case "info":
		l = zerolog.InfoLevel
	case "debug":
		l = zerolog.DebugLevel
	default:
		l = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(l)

	// Pretty console output
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	logger := zerolog.New(output).
		Level(l).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{
		logger: &logger,
	}
}

// Debug -.
func (l *Logger) Debug(message interface{}, args ...interface{}) {
	l.msg("debug", message, args...)
}

// Info -.
func (l *Logger) Info(message interface{}, args ...interface{}) {
	l.msg("info", message, args...)
}

// Warn -.
func (l *Logger) Warn(message interface{}, args ...interface{}) {
	l.msg("warn", message, args...)
}

// Error -.
func (l *Logger) Error(message interface{}, args ...interface{}) {
	l.msg("error", message, args...)
}

// Fatal -.
func (l *Logger) Fatal(message interface{}, args ...interface{}) {
	l.msg("fatal", message, args...)

	os.Exit(1)
}

func (l *Logger) msg(level string, message interface{}, args ...interface{}) {
	switch msg := message.(type) {
	case error:
		l.log(level, msg.Error(), args...)
	case string:
		l.log(level, msg, args...)
	default:
		l.log(level, fmt.Sprintf("%v", message), args...)
	}
}

func (l *Logger) log(level string, message string, args ...interface{}) {
	if len(args) == 0 {
		switch level {
		case "debug":
			l.logger.Debug().Msg(message)
		case "info":
			l.logger.Info().Msg(message)
		case "warn":
			l.logger.Warn().Msg(message)
		case "error":
			l.logger.Error().Msg(message)
		case "fatal":
			l.logger.Fatal().Msg(message)
		}
		return
	}

	// Handle key-value pairs
	var event *zerolog.Event
	switch level {
	case "debug":
		event = l.logger.Debug()
	case "info":
		event = l.logger.Info()
	case "warn":
		event = l.logger.Warn()
	case "error":
		event = l.logger.Error()
	case "fatal":
		event = l.logger.Fatal()
	default:
		event = l.logger.Info()
	}

	// Add fields from args (key-value pairs)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			event = event.Interface(key, args[i+1])
		}
	}

	event.Msg(message)
}

// WithField - добавить одно поле к логгеру
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := l.logger.With().Interface(key, value).Logger()
	return &Logger{logger: &newLogger}
}

// WithFields - добавить несколько полей к логгеру
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	newLogger := ctx.Logger()
	return &Logger{logger: &newLogger}
}

// GetZerolog - получить нативный zerolog.Logger для продвинутого использования
func (l *Logger) GetZerolog() *zerolog.Logger {
	return l.logger
}
