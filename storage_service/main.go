package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Contract represents the stored contract data
type Contract struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Parameters json.RawMessage `json:"parameters"`
	CreatedAt  int64           `json:"created_at"`
	IsActive   bool            `json:"is_active"`
	Duration   int             `json:"duration"`
}

// Storage interface defines the persistence operations
type Storage interface {
	Save(id string, contract *Contract) error
	Get(id string) (*Contract, error)
	Delete(id string) error
	GetAll() ([]*Contract, error)
	Clean() error
}

// PostgresStorage implements Storage interface for PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(host, port, user, password, dbname string) (*PostgresStorage, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	log.Printf("Connecting to database with: host=%s port=%s user=%s dbname=%s",
		host, port, user, dbname)

	// Try to connect with retries
	var db *sql.DB
	var err error
	for i := 0; i < 30; i++ {
		log.Printf("Attempting database connection (attempt %d/30)...", i+1)
		db, err = sql.Open("postgres", psqlInfo)
		if err == nil {
			log.Printf("Successfully opened database, attempting ping...")
			err = db.Ping()
			if err == nil {
				log.Printf("Successfully pinged database")
				break
			}
			log.Printf("Ping failed: %v", err)
		} else {
			log.Printf("Open failed: %v", err)
		}
		log.Printf("Failed to connect to database, retrying in 2 seconds... (attempt %d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after 30 attempts: %v", err)
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Clean() error {
	_, err := s.db.Exec("DELETE FROM contracts")
	return err
}

func (s *PostgresStorage) Save(id string, contract *Contract) error {
	_, err := s.db.Exec(`
		INSERT INTO contracts (id, type, parameters, created_at, is_active, duration)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			parameters = EXCLUDED.parameters,
			created_at = EXCLUDED.created_at,
			is_active = EXCLUDED.is_active,
			duration = EXCLUDED.duration
	`, contract.ID, contract.Type, contract.Parameters, contract.CreatedAt, contract.IsActive, contract.Duration)
	return err
}

func (s *PostgresStorage) Get(id string) (*Contract, error) {
	var contract Contract
	var parameters []byte
	err := s.db.QueryRow(`
		SELECT id, type, parameters, created_at, is_active, duration
		FROM contracts WHERE id = $1
	`, id).Scan(&contract.ID, &contract.Type, &parameters, &contract.CreatedAt, &contract.IsActive, &contract.Duration)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	contract.Parameters = json.RawMessage(parameters)
	return &contract, nil
}

func (s *PostgresStorage) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM contracts WHERE id = $1", id)
	return err
}

func (s *PostgresStorage) GetAll() ([]*Contract, error) {
	rows, err := s.db.Query(`
		SELECT id, type, parameters, created_at, is_active, duration
		FROM contracts
	`)
	if err != nil {
		return make([]*Contract, 0), nil // Return empty slice instead of nil
	}
	defer rows.Close()

	var contracts []*Contract
	for rows.Next() {
		var contract Contract
		var parameters []byte
		err := rows.Scan(&contract.ID, &contract.Type, &parameters, &contract.CreatedAt, &contract.IsActive, &contract.Duration)
		if err != nil {
			return make([]*Contract, 0), nil // Return empty slice instead of nil
		}
		contract.Parameters = json.RawMessage(parameters)
		contracts = append(contracts, &contract)
	}
	if contracts == nil {
		return make([]*Contract, 0), nil // Return empty slice instead of nil
	}
	return contracts, nil
}

type server struct {
	storage Storage
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Try to ping the database
	if s.storage != nil {
		if db, ok := s.storage.(*PostgresStorage); ok {
			if err := db.db.Ping(); err != nil {
				log.Printf("Health check failed: %v", err)
				http.Error(w, fmt.Sprintf("Database not healthy: %v", err), http.StatusServiceUnavailable)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *server) handleSaveContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var contract Contract
	if err := json.NewDecoder(r.Body).Decode(&contract); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.storage.Save(contract.ID, &contract); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *server) handleGetContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		contracts, err := s.storage.GetAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if contracts == nil {
			contracts = make([]*Contract, 0) // Return empty slice instead of nil
		}
		if err := json.NewEncoder(w).Encode(contracts); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	contract, err := s.storage.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if contract == nil {
		http.Error(w, "Contract not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(contract); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) handleDeleteContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := s.storage.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *server) handleCleanDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.storage.Clean(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	log.Printf("Starting storage service...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	log.Printf("Database configuration: host=%s port=%s user=%s dbname=%s",
		dbHost, dbPort, dbUser, dbName)

	storage, err := NewPostgresStorage(dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	srv := &server{storage: storage}

	http.HandleFunc("/health", srv.handleHealth)
	http.HandleFunc("/contract", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			srv.handleSaveContract(w, r)
		case http.MethodGet:
			srv.handleGetContract(w, r)
		case http.MethodDelete:
			srv.handleDeleteContract(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/clean", srv.handleCleanDB)

	log.Printf("Storage service HTTP server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
