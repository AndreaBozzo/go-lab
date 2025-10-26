package storage

import "github.com/AndreaBozzo/go-logboard/internal/collector"

type LogStorage interface {
	Save(logs []collector.LogEntry) error
	QueryLogs(limit int) ([]collector.LogEntry, error)
}
