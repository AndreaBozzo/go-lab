/*
internal/storage/sqlite.go
Package storage provides SQLite storage implementation for log entries.
*/

package storage

import (
	"database/sql"
	"time"

	"github.com/AndreaBozzo/go-lab/internal/collector"
	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dataSourceName string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Create logs table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT,
		level TEXT,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		method TEXT,
		path TEXT,
		status_code INTEGER,
		latency_ms INTEGER,
		client_ip TEXT,
		user_agent TEXT,
		backend TEXT
	)`)
	if err != nil {
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

var _ LogStorage = (*SQLiteStorage)(nil)

func (s *SQLiteStorage) SaveLog(entry collector.LogEntry) error {
	_, err := s.db.Exec(`INSERT INTO logs
		(source, level, message, timestamp, method, path, status_code, latency_ms, client_ip, user_agent, backend)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Source, entry.Level, entry.Message, entry.Time,
		entry.Method, entry.Path, entry.StatusCode, entry.Latency.Milliseconds(),
		entry.ClientIP, entry.UserAgent, entry.Backend)
	return err
}

func (s *SQLiteStorage) Save(logs []collector.LogEntry) error {
	for _, entry := range logs {
		if err := s.SaveLog(entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) QueryLogs(limit int) ([]collector.LogEntry, error) {
	rows, err := s.db.Query(`SELECT source, level, message, timestamp, method, path, status_code, latency_ms, client_ip, user_agent, backend
		FROM logs ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []collector.LogEntry
	for rows.Next() {
		var entry collector.LogEntry
		var latencyMs int64
		if err := rows.Scan(&entry.Source, &entry.Level, &entry.Message, &entry.Time,
			&entry.Method, &entry.Path, &entry.StatusCode, &latencyMs,
			&entry.ClientIP, &entry.UserAgent, &entry.Backend); err != nil {
			return nil, err
		}
		entry.Latency = time.Duration(latencyMs) * time.Millisecond
		results = append(results, entry)
	}
	return results, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
