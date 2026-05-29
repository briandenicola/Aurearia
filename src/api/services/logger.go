package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	LevelTrace LogLevel = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[LogLevel]string{
	LevelTrace: "TRACE",
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

var levelFromString = map[string]LogLevel{
	"trace": LevelTrace,
	"debug": LevelDebug,
	"info":  LevelInfo,
	"warn":  LevelWarn,
	"error": LevelError,
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type Logger struct {
	mu       sync.RWMutex
	level    LogLevel
	buffer   []LogEntry
	maxSize  int
	writeIdx int
	count    int
}

func NewLogger(bufferSize int) *Logger {
	return &Logger{
		level:   LevelInfo,
		buffer:  make([]LogEntry, bufferSize),
		maxSize: bufferSize,
	}
}

func (l *Logger) SetLevel(levelStr string) {
	if lvl, ok := levelFromString[strings.ToLower(levelStr)]; ok {
		l.mu.Lock()
		l.level = lvl
		l.mu.Unlock()
		l.Info("logger", "Log level changed to %s", strings.ToUpper(levelStr))
	}
}

func (l *Logger) GetLevel() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return levelNames[l.level]
}

func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

func (l *Logger) write(level LogLevel, component, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	msg := fmt.Sprintf(format, args...)
	levelName := levelNames[level]
	now := time.Now().UTC().Format(time.RFC3339)

	fullMsg := fmt.Sprintf("[%s] %s", component, msg)

	// Write to console via standard log
	log.Printf("[%s] %s", levelName, fullMsg)

	// Write to ring buffer
	entry := LogEntry{
		Timestamp: now,
		Level:     levelName,
		Message:   fullMsg,
	}

	l.mu.Lock()
	l.buffer[l.writeIdx] = entry
	l.writeIdx = (l.writeIdx + 1) % l.maxSize
	if l.count < l.maxSize {
		l.count++
	}
	l.mu.Unlock()
}

func (l *Logger) Trace(component, format string, args ...interface{}) {
	l.write(LevelTrace, component, format, args...)
}

func (l *Logger) Debug(component, format string, args ...interface{}) {
	l.write(LevelDebug, component, format, args...)
}

func (l *Logger) Info(component, format string, args ...interface{}) {
	l.write(LevelInfo, component, format, args...)
}

func (l *Logger) Warn(component, format string, args ...interface{}) {
	l.write(LevelWarn, component, format, args...)
}

func (l *Logger) Error(component, format string, args ...interface{}) {
	l.write(LevelError, component, format, args...)
}

// GetLogs returns the most recent log entries, ordered oldest-first
func (l *Logger) GetLogs(limit int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > l.count {
		limit = l.count
	}

	result := make([]LogEntry, 0)

	// Calculate start index: oldest entry within the requested limit
	start := (l.writeIdx - limit + l.maxSize) % l.maxSize
	for i := 0; i < limit; i++ {
		idx := (start + i) % l.maxSize
		result = append(result, l.buffer[idx])
	}

	return result
}

// (SyncLogLevel moved to SettingsService.SyncLogLevel)
