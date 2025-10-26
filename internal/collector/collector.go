/*
internal/collector/collector.go
Package collector provides log collection capabilities from various sources.
*/

package collector

import (
	"fmt"
	"time"
)

type LogEntry struct {
	Source  string
	Level   string
	Message string
	Time    time.Time

	// HTTP-specific fields for API Gateway logging
	Method     string
	Path       string
	StatusCode int
	Latency    time.Duration
	ClientIP   string
	UserAgent  string
	Backend    string // Backend server that handled the request
}

type Collector interface {
	Collect() ([]LogEntry, error)
}

type FileCollector struct {
	Path string
}

func (f *FileCollector) Collect() ([]LogEntry, error) {
	fmt.Println("Reading logs from", f.Path)
	// TODO: implementa parsing file
	return []LogEntry{
		{Source: f.Path, Message: "Example log", Level: "INFO", Time: time.Now()},
	}, nil
}
