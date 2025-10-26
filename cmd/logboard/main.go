package main

import (
	"fmt"
	"log"

	"github.com/AndreaBozzo/go-logboard/internal/collector"
	"github.com/AndreaBozzo/go-logboard/internal/storage"
)

func main() {
	// Initialize components
	fmt.Println("Starting Logboard...")

	store, err := storage.NewSQLiteStorage("data.db")
	if err != nil {
		log.Fatal("Error initializing storage:", err)
	}
	defer store.Close()

	// Create a file collector
	fileCollector := &collector.FileCollector{Path: "logs.txt"}

	logs, err := fileCollector.Collect()
	if err != nil {
		log.Fatal("Error collecting logs:", err)
	}

	// Store collected logs
	store.SaveLogs(logs)
}
