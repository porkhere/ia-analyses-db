package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"ia-analyses-db/internal/config"
)

func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", BuildDSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	return db, nil
}
