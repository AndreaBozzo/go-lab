/*
internal/storage/sqlite.go
Package storage provides SQLite storage implementation for log entries.
*/

package storage

import (
	"database/sql"

	"github.com/AndreaBozzo/go-lab/internal/collector"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dataSourceName string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Create logs table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		level TEXT,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

var _ LogStorage = (*SQLiteStorage)(nil)

func (s *SQLiteStorage) SaveLog(entry collector.LogEntry) error {
	_, err := s.db.Exec("INSERT INTO logs (level, message, timestamp) VALUES (?, ?, ?)",
		entry.Level, entry.Message, entry.Time)
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
	rows, err := s.db.Query("SELECT level, message, timestamp FROM logs ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []collector.LogEntry
	for rows.Next() {
		var entry collector.LogEntry
		if err := rows.Scan(&entry.Level, &entry.Message, &entry.Time); err != nil {
			return nil, err
		}
		results = append(results, entry)
	}
	return results, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
