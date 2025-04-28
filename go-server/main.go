package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Configuration for database connections
type Config struct {
	PrimaryDSN string
	ReplicaDSN string
}

// User represents a user in our system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	primaryDB *sql.DB
	replicaDB *sql.DB
)

func main() {
	// Load configuration
	config := Config{
		PrimaryDSN: "postgres://postgres:password@localhost:5432/appdb?sslmode=disable",
		ReplicaDSN: "postgres://postgres:password@localhost:5433/appdb?sslmode=disable",
	}

	// Initialize database connections
	err := initDBConnections(config)
	if err != nil {
		log.Fatalf("Failed to initialize database connections: %v", err)
	}
	defer primaryDB.Close()
	defer replicaDB.Close()

	// Ensure tables exist
	err = ensureTablesExist()
	if err != nil {
		log.Fatalf("Failed to ensure tables exist: %v", err)
	}

	// Set up router
	r := mux.NewRouter()
	r.HandleFunc("/users", createUserHandler).Methods("POST")
	r.HandleFunc("/users", getUsersHandler).Methods("GET")
	r.HandleFunc("/users/{id}", getUserHandler).Methods("GET")
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func initDBConnections(config Config) error {
	var err error

	// Connect to primary database
	primaryDB, err = sql.Open("postgres", config.PrimaryDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to primary database: %v", err)
	}
	
	// Test primary connection
	err = primaryDB.Ping()
	if err != nil {
		return fmt.Errorf("primary database ping failed: %v", err)
	}
	log.Println("Connected to primary database")

	// Connect to replica database
	replicaDB, err = sql.Open("postgres", config.ReplicaDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to replica database: %v", err)
	}
	
	// Test replica connection
	err = replicaDB.Ping()
	if err != nil {
		return fmt.Errorf("replica database ping failed: %v", err)
	}
	log.Println("Connected to replica database")

	// Configure connection pools
	primaryDB.SetMaxOpenConns(25)
	primaryDB.SetMaxIdleConns(5)
	primaryDB.SetConnMaxLifetime(5 * time.Minute)
	
	replicaDB.SetMaxOpenConns(25)
	replicaDB.SetMaxIdleConns(5)
	replicaDB.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

func ensureTablesExist() error {
	// Create users table if it doesn't exist
	_, err := primaryDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	
	return err
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	
	// Decode request body
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Insert user into database (using primary)
	err := insertUser(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Get users from database (using replica for read)
	users, err := getUsers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL parameters
	vars := mux.Vars(r)
	id := vars["id"]

	// Get user from database (using replica for read)
	user, err := getUser(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check primary connection
	primaryErr := primaryDB.Ping()
	
	// Check replica connection
	replicaErr := replicaDB.Ping()
	
	status := map[string]string{
		"primary": "up",
		"replica": "up",
	}
	
	httpStatus := http.StatusOK
	
	if primaryErr != nil {
		status["primary"] = "down: " + primaryErr.Error()
		httpStatus = http.StatusServiceUnavailable
	}
	
	if replicaErr != nil {
		status["replica"] = "down: " + replicaErr.Error()
		httpStatus = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(status)
}

// Database operations

func insertUser(user User) error {
	query := `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, created_at
	`
	
	return primaryDB.QueryRow(query, user.Name, user.Email).Scan(&user.ID, &user.CreatedAt)
}

func getUsers() ([]User, error) {
	query := `
		SELECT id, name, email, created_at
		FROM users
		ORDER BY id
	`
	
	rows, err := replicaDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

func getUser(id string) (User, error) {
	query := `
		SELECT id, name, email, created_at
		FROM users
		WHERE id = $1
	`
	
	var user User
	err := replicaDB.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	return user, err
}