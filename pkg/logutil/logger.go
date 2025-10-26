package logutil

import (
	"fmt"
	"log"
	"time"

	"github.com/AndreaBozzo/go-lab/internal/collector"
	"github.com/AndreaBozzo/go-lab/internal/storage"
)

type logWriter struct {
	storage storage.LogStorage
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	entry := collector.LogEntry{
		Source:  "logger", // Add source identification
		Level:   "INFO",   // Set appropriate log level
		Message: string(p),
		Time:    time.Now(), // Use 'Time' instead of 'Timestamp'
	}

	if err := w.storage.Save([]collector.LogEntry{entry}); err != nil {
		return 0, fmt.Errorf("failed to save log entry: %w", err)
	}

	return len(p), nil
}

// NewLogger creates a new logger that writes to the provided LogStorage.
func NewLogger(storage storage.LogStorage) *log.Logger {
	writer := &logWriter{storage: storage}
	return log.New(writer, "", 0)
}
