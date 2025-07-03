package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

type ZerologAdapter struct {
	logger zerolog.Logger
}

func NewZerolog(writer io.Writer, level zerolog.Level) *ZerologAdapter {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.DurationFieldInteger = true

	logger := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &ZerologAdapter{logger: logger}
}

func NewConsoleLogger(level zerolog.Level) *ZerologAdapter {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	}
	return NewZerolog(consoleWriter, level)
}

func (z *ZerologAdapter) Info(component, message string, fields map[string]interface{}) {
	if !z.logger.Info().Enabled() {
		return
	}

	event := z.logger.Info().Str("component", component)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(message)
}

func (z *ZerologAdapter) Error(component string, err error, fields map[string]interface{}) {
	if !z.logger.Error().Enabled() {
		return
	}

	event := z.logger.Error().Str("component", component).Err(err)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg("operation failed")
}

func (z *ZerologAdapter) Warning(component, message string, fields map[string]interface{}) {
	if !z.logger.Warn().Enabled() {
		return
	}

	event := z.logger.Warn().Str("component", component)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(message)
}

func (z *ZerologAdapter) Debug(component, message string, fields map[string]interface{}) {
	if !z.logger.Debug().Enabled() {
		return
	}

	event := z.logger.Debug().Str("component", component)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(message)
}
