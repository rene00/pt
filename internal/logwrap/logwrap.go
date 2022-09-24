// Package logwrap is a wrapper around stdlib log that provides log levels.
//
// Copied from https://codereview.stackexchange.com/questions/272990/go-logging-wrapper-that-adds-log-levels
//
package logwrap

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	// DEBUG for debugging.
	DEBUG = 0
	// INFO for info messages.
	INFO = 10
	// WARNING for warning messages.
	WARNING = 20
	// ERROR for error messages.
	ERROR = 30
	// NONE for no messages.
	NONE = 100
)

var (
	pkgLock sync.Mutex
	allLogs = make(map[string]*LogWrap)
)

// LogWrap is a struct.
type LogWrap struct {
	name         string
	level        int
	debugLogWrap *log.Logger
	infoLogWrap  *log.Logger
	logfileinfo  bool
	lock         sync.Mutex
}

// New is the constructor for logger.
func New(name string, dest io.Writer, logfileinfo bool, flags ...int) *LogWrap {
	pkgLock.Lock()
	defer pkgLock.Unlock()
	if _, exists := allLogs[name]; exists {
		panic(fmt.Sprintf("Unable to create logger with %s: name already in use", name))
	}

	var logFlags int
	if flags == nil {
		logFlags = log.Ldate | log.Ltime | log.Lmsgprefix
	} else {
		logFlags = flags[0]
	}

	logger := &LogWrap{
		name:         name,
		debugLogWrap: log.New(dest, "DEBUG: ", logFlags),
		infoLogWrap:  log.New(dest, "INFO: ", logFlags),
	}

	allLogs[name] = logger
	return logger
}

// getfileinfo gets the calling filename and line number.
func getfileinfo() string {
	_, filename, line, ok := runtime.Caller(2)
	if !ok {
		filename = "Unknown"
		line = 0
	}
	return fmt.Sprintf("%s:%d: ", filepath.Base(filename), line)
}

// SetLevel sets the log level for the respective LogWrap. Each level is an int
// that can be set via pkg consts (eg; logger.DEBUG) or via literal ints.
// Messages sent to LogWraps with a log level lower than the current level are
// not written.
func (l *LogWrap) SetLevel(level int) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.level = level
}

// Info prints info messages.
func (l *LogWrap) Info(msg string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.level > INFO {
		return
	}
	if l.logfileinfo {
		msg = getfileinfo() + msg
	}
	l.infoLogWrap.Println(msg)
}

// Debug prints debug messages.
func (l *LogWrap) Debug(msg string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.level > DEBUG {
		return
	}
	if l.logfileinfo {
		msg = getfileinfo() + msg
	}
	l.infoLogWrap.Println(msg)
}

// Get returns a reference to an existing LogWrap if one exists, othewise nil.
func Get(name string) *LogWrap {
	pkgLock.Lock()
	defer pkgLock.Unlock()
	if foundLog, ok := allLogs[name]; ok {
		return foundLog
	}
	return nil
}
