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
