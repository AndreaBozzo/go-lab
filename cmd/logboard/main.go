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
)

func main() {
	// Initialize components
	fmt.Println("Starting Logboard...")

	var store storage.LogStorage
	sqliteStore, err := storage.NewSQLiteStorage("data.db")
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}
	defer sqliteStore.Close()
	store = sqliteStore

	fileCollector := &collector.FileCollector{Path: "logs.txt"}
	logs, err := fileCollector.Collect()
	if err != nil {
		log.Fatal("Failed to collect logs:", err)
	}

	if err := store.Save(logs); err != nil {
		log.Fatal("Failed to save logs:", err)
	}

	fmt.Println("Logs saved successfully!")
}
