package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	levelWidth   = 5
	callerWidth  = 35
	messageWidth = 40
)

// newEncoder selects console output only for the explicit development env.
//
// All other values, including an empty env, fall back to JSON output.
func newEncoder(env string, colorize bool) zapcore.Encoder {
	if env == "development" {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		cfg.EncodeLevel = fixedWidthLevelEncoder(colorize)
		cfg.EncodeCaller = fixedWidthCallerEncoder
		cfg.ConsoleSeparator = "  "

		return zapcore.NewConsoleEncoder(cfg)
	}

	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewJSONEncoder(cfg)
}

type messagePaddingCore struct {
	zapcore.Core
	width int
}

func (core *messagePaddingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !core.Enabled(entry.Level) {
		return checked
	}

	return checked.AddCore(entry, core)
}

func (core *messagePaddingCore) With(fields []zapcore.Field) zapcore.Core {
	return &messagePaddingCore{
		Core:  core.Core.With(fields),
		width: core.width,
	}
}

func (core *messagePaddingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	entry.Message = padString(entry.Message, core.width) + " "
	return core.Core.Write(entry, fields)
}

func fixedWidthLevelEncoder(colorize bool) zapcore.LevelEncoder {
	return func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		label := padString(level.CapitalString(), levelWidth)
		if !colorize {
			enc.AppendString(label)
			return
		}

		enc.AppendString(colorizeLevel(level, label))
	}
}

func fixedWidthCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(padString(caller.TrimmedPath(), callerWidth))
}

func colorizeLevel(level zapcore.Level, label string) string {
	switch level {
	case zapcore.DebugLevel:
		return "\x1b[35m" + label + "\x1b[0m"
	case zapcore.InfoLevel:
		return "\x1b[34m" + label + "\x1b[0m"
	case zapcore.WarnLevel:
		return "\x1b[33m" + label + "\x1b[0m"
	case zapcore.ErrorLevel:
		return "\x1b[31m" + label + "\x1b[0m"
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return "\x1b[31m" + label + "\x1b[0m"
	default:
		return label
	}
}

func padString(value string, width int) string {
	if len(value) >= width {
		return value
	}

	return value + strings.Repeat(" ", width-len(value))
}
