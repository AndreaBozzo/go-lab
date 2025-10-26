/*
internal/storage/storage.go
Package storage provides an interface for log storage implementations.
*/

package storage

import "github.com/AndreaBozzo/go-lab/internal/collector"

type LogStorage interface {
	Save(logs []collector.LogEntry) error
	QueryLogs(limit int) ([]collector.LogEntry, error)
}
