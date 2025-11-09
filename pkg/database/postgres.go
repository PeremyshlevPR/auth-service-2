package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Postgres represents a PostgreSQL database connection
type Postgres struct {
	DB *sql.DB
}

// NewPostgres creates a new PostgreSQL connection
func NewPostgres(dsn string) (*Postgres, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Postgres{DB: db}, nil
}

// Close closes the database connection
func (p *Postgres) Close() error {
	return p.DB.Close()
}

// Ping checks if the database is available
func (p *Postgres) Ping() error {
	return p.DB.Ping()
}
