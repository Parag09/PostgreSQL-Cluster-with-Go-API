package main

import (
	"database/sql"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	primaryConnStr = "postgres://postgres:password@localhost:5432/appdb?sslmode=disable"
	replicaConnStr = "postgres://postgres:password@localhost:5433/appdb?sslmode=disable"
)

func main() {
	// Connect to the primary database
	primaryDB, err := sql.Open("postgres", primaryConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to primary: %v", err)
	}
	defer primaryDB.Close()

	// Create test table
	_, err = primaryDB.Exec(`
		CREATE TABLE IF NOT EXISTS deadlock_test (
			id SERIAL PRIMARY KEY,
			value TEXT
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Insert initial rows
	_, err = primaryDB.Exec(`
		INSERT INTO deadlock_test (value) VALUES ('A'), ('B')
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		log.Fatalf("Failed to insert initial rows: %v", err)
	}

	// Simulate deadlock
	simulateDeadlock(primaryConnStr)
}

func simulateDeadlock(connStr string) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		session1(connStr)
	}()

	go func() {
		defer wg.Done()
		session2(connStr)
	}()

	wg.Wait()
}

func session1(connStr string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Session1: Failed to connect: %v", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Session1: Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Lock row 1
	_, err = tx.Exec("UPDATE deadlock_test SET value = 'Session1' WHERE id = 1")
	if err != nil {
		log.Fatalf("Session1: Failed to lock row 1: %v", err)
	}

	time.Sleep(2 * time.Second) // Simulate delay

	// Try to lock row 2
	_, err = tx.Exec("UPDATE deadlock_test SET value = 'Session1' WHERE id = 2")
	if err != nil {
		log.Printf("Session1: Deadlock detected: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Session1: Failed to commit: %v", err)
	}
}

func session2(connStr string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Session2: Failed to connect: %v", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Session2: Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Lock row 2
	_, err = tx.Exec("UPDATE deadlock_test SET value = 'Session2' WHERE id = 2")
	if err != nil {
		log.Fatalf("Session2: Failed to lock row 2: %v", err)
	}

	time.Sleep(2 * time.Second) // Simulate delay

	// Try to lock row 1
	_, err = tx.Exec("UPDATE deadlock_test SET value = 'Session2' WHERE id = 1")
	if err != nil {
		log.Printf("Session2: Deadlock detected: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Session2: Failed to commit: %v", err)
	}
}
