package audit

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	level  string
	output *log.Logger
	file   *os.File
}

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

func NewLogger(level string) *Logger {
	// Create logs directory if not exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Warning: Failed to create logs directory: %v", err)
	}

	// Open log file
	logFile, err := os.OpenFile("logs/agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: Failed to open log file: %v", err)
	}

	// Multi-writer for console and file
	multiWriter := log.New(os.Stdout, "", 0)

	return &Logger{
		level:  strings.ToUpper(level),
		output: multiWriter,
		file:   logFile,
	}
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if !l.shouldLog(string(level)) {
		return
	}

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Get caller info
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	funcName = funcName[strings.LastIndex(funcName, ".")+1:]

	// Shorten file path
	fileParts := strings.Split(file, "/")
	fileName := fileParts[len(fileParts)-1]

	logMessage := fmt.Sprintf("[%s] %s | %s:%d %s() | %s",
		level, timestamp, fileName, line, funcName, msg)

	l.output.Println(logMessage)

	if l.file != nil {
		l.file.WriteString(logMessage + "\n")
	}
}

func (l *Logger) shouldLog(level string) bool {
	levels := map[string]int{
		"DEBUG": 0,
		"INFO":  1,
		"WARN":  2,
		"ERROR": 3,
	}

	current, ok := levels[l.level]
	if !ok {
		current = 1
	}

	target, ok := levels[level]
	if !ok {
		return true
	}

	return target >= current
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
