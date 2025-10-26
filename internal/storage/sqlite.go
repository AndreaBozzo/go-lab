package storage

import (
	"database/sql"

	"github.com/AndreaBozzo/go-logboard/internal/collector"
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

func (s *SQLiteStorage) SaveLog(entry collector.LogEntry) error {
	_, err := s.db.Exec(`INSERT INTO logs (level, message) VALUES (?, ?)`, entry.Level, entry.Message)
	return err

}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
