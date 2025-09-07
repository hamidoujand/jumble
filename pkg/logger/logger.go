// Package logger provides a custom logger with trace id support and more.
package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"
)

// TraceIDFn knows how to extract trace id from the context passed to it.
// client of this package need to implement that logic.
type TraceIDFn func(ctx context.Context) string

// Level represent the logging levels used by logger, we define this so client
// is abstracted from slog.Level and we have flexibility to change in future.
type Level slog.Level

const (
	LevelDebug = Level(slog.LevelDebug)
	LevelInfo  = Level(slog.LevelInfo)
	LevelWarn  = Level(slog.LevelWarn)
	LevelError = Level(slog.LevelError)
)

// Environment represents the environment that logger being used.
type Environment int

const (
	EnvironmentDev  Environment = 1
	EnvironmentProd Environment = 2
)

// Logger represents a logger with a custom handler to log information.
type Logger struct {
	//not using embedding since the relation between logger and handler follows
	//this pattern (logger HAS A handler) so "composition as field" is better choice.
	handler slog.Handler

	//discard allows the logger to skip the logging.
	discard bool

	//traceIDFn extract the traceID from ctx.
	traceIDFn TraceIDFn
}

// New creates a logger and returns it.
func New(w io.Writer, minLevel Level, env Environment, serviceName string, traceIDFn TraceIDFn) Logger {
	handler := createHandler(w, serviceName, minLevel, env)
	return Logger{
		handler:   handler,
		discard:   w == io.Discard,
		traceIDFn: traceIDFn,
	}
}

// Debug logs at the LevelDebug.
func (l Logger) Debug(ctx context.Context, msg string, args ...any) {
	if l.discard {
		return
	}

	//frame 0: runtime.Callers()
	//frame 1: write()
	//frame 2: Debug()
	//frame 3: debug caller (what we want) so we skip those 3
	l.write(ctx, LevelDebug, 3, msg, args...)
}

// Debugc logs at the LevelDebug and callstack position caller.
func (l Logger) Debugc(ctx context.Context, caller int, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelDebug, caller, msg, args...)
}

// Info logs at the LevelInfo.
func (l Logger) Info(ctx context.Context, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelInfo, 3, msg, args...)
}

// Infoc logs at the LevelInfo and callstack position caller.
func (l Logger) Infoc(ctx context.Context, caller int, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelInfo, caller, msg, args...)
}

// Warn logs at the LevelWarn.
func (l Logger) Warn(ctx context.Context, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelWarn, 3, msg, args...)
}

// Warnc logs at the LevelWarn and callstack position caller.
func (l Logger) Warnc(ctx context.Context, caller int, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelWarn, caller, msg, args...)
}

// Error logs at the LevelError.
func (l Logger) Error(ctx context.Context, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelError, 3, msg, args...)
}

// Errorc logs at the LevelError and callstack position caller.
func (l Logger) Errorc(ctx context.Context, caller int, msg string, args ...any) {
	if l.discard {
		return
	}

	l.write(ctx, LevelError, caller, msg, args...)
}

// NewStdLogger takes a Logger and returns a standard logger can be used in http.server to log error messages.
func NewStdLogger(log Logger, level Level) *log.Logger {
	return slog.NewLogLogger(log.handler, slog.Level(level))
}

func (l Logger) write(ctx context.Context, level Level, skipStack int, msg string, args ...any) {
	//check to see if log is enabled for this level
	slogLevel := slog.Level(level)
	if !l.handler.Enabled(ctx, slogLevel) {
		return
	}

	//skip callstack

	var pcs [1]uintptr //uintptr is a just an integer large  enough to hold a memory address.

	//each integer inside that array is a memory address to executable instruction inside compiled Go program.
	//array of 1, because we need the immediate caller (who called me) and we also skip few frames as well.
	//skip starts from top to bottom and start skipping stacks the top most one is always "runtime.Callers()" itself
	//and skip=1 means only skip "runtime.Caller()" and then give me the "immediate stack pointer"
	/*
			Skip 0: runtime.Callers (useless)

		    Skip 1: Your current function (where you called runtime.Callers)

		    Skip 2: The function that called you (what you usually want)

		    Skip 3: The function that called your caller
	*/
	runtime.Callers(skipStack, pcs[:])

	//create a log record
	logRecord := slog.NewRecord(time.Now(), slog.Level(level), msg, pcs[0])

	if l.traceIDFn != nil {
		args = append(args, "traceID", l.traceIDFn(ctx))
	}

	//add args to the record
	logRecord.Add(args...)

	l.handler.Handle(ctx, logRecord)
}

//==============================================================================

func createHandler(w io.Writer, service string, minLevel Level, env Environment) slog.Handler {
	//custom file name
	fn := func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.SourceKey {
			source, ok := attr.Value.Any().(*slog.Source)
			if !ok {
				return attr
			}

			filename := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
			return slog.Attr{Key: "file", Value: slog.StringValue(filename)}
		}

		return attr
	}

	//create handler
	var handler slog.Handler

	if env == EnvironmentProd {
		//json handler
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: true, Level: slog.Level(minLevel), ReplaceAttr: fn})
	} else {
		//text handler
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{AddSource: true, Level: slog.Level(minLevel), ReplaceAttr: fn})
	}

	//adding default attrs
	attrs := []slog.Attr{
		{Key: "service", Value: slog.StringValue(service)},
	}

	handler = handler.WithAttrs(attrs)
	return handler
}
