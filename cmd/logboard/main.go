/*
cmd/logboard/main.go
Logboard is a simple logging application that collects logs from various sources and stores them in a SQLite database.
*/

package main

import (
	"fmt"
	"log"

	"github.com/AndreaBozzo/go-lab/internal/collector"
	"github.com/AndreaBozzo/go-lab/internal/storage"
	"github.com/AndreaBozzo/go-lab/pkg/logutil"
)

func main() {
	// Initialize storage
	log.Println("Starting Logboard...")

	// Initialize SQLite storage
	sqliteStore, err := storage.NewSQLiteStorage("data.db")
	if err != nil {
		log.Fatalf("Failed to initialize SQLite storage: %v", err)
	}
	defer sqliteStore.Close()

	// Initialize persistent log
	logger := logutil.NewLogger(sqliteStore)
	logger.Println("Logboard initialized successfully.")

	// Create file collector
	fileCollector := &collector.FileCollector{Path: "logs.txt"}
	logs, err := fileCollector.Collect()
	if err != nil {
		logger.Printf("Failed to collect logs: %v", err)
		return
	}

	// Store collected logs
	if err := sqliteStore.Save(logs); err != nil {
		logger.Fatalf("Failed to store logs: %v", err)
	}

	logger.Printf("Successfully collected and stored %d logs.", len(logs))
	fmt.Println("Logboard operation completed.")
}
