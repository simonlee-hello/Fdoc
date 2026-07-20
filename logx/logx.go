package logx

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	infoLog  = log.New(os.Stdout, "", log.LstdFlags)
	warnLog  = log.New(os.Stderr, "", log.LstdFlags)
	errorLog = log.New(os.Stderr, "", log.LstdFlags)
	debugLog = log.New(io.Discard, "", log.LstdFlags)
)

// SetQuiet suppresses info output (warnings/errors still print).
func SetQuiet(quiet bool) {
	if quiet {
		infoLog.SetOutput(io.Discard)
	} else {
		infoLog.SetOutput(os.Stdout)
	}
}

// SetDebug enables debug logs on stderr.
func SetDebug(enabled bool) {
	if enabled {
		debugLog.SetOutput(os.Stderr)
	} else {
		debugLog.SetOutput(io.Discard)
	}
}

func Info(format string, args ...any) {
	infoLog.Output(2, fmt.Sprintf(format, args...))
}

func Warning(format string, args ...any) {
	warnLog.Output(2, fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	errorLog.Output(2, fmt.Sprintf(format, args...))
}

func Debug(format string, args ...any) {
	debugLog.Output(2, fmt.Sprintf(format, args...))
}
