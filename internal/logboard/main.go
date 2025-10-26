package main

import (
	"fmt"
	"log"

	"github.com/AndreaBozzo/go-lab/internal/collector"
	"github.com/AndreaBozzo/go-lab/internal/storage"
)

func main() {
	// Initialize components
	fmt.Println("Starting Logboard...")

	collector := &collector.FileCollector{Path: "logs.txt"}
	logs, _ := collector.Collect()

	store, err := storage.NewSQLiteStorage("data.db")
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}
	defer store.Close()

	// Process logs
	for _, logEntry := range logs {
		if err := store.SaveLog(logEntry); err != nil {
			log.Printf("Error saving log entry: %v", err)
		}
	}

	fmt.Println("Logboard started successfully.")
}
