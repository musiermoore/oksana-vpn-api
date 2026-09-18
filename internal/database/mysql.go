package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)

	// Verify that MySQL is reachable
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to mysql: %w", err)
	}

	return db, nil
}

func BuildDSN(user string, psw string, hostname string, port string, database string) (string, error) {
	if user == "" && psw == "" && hostname == "" && port == "" && database == "" {
		return "", fmt.Errorf("Invalid params to build DSN")
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=5s",
		user, psw, hostname, port, database,
	), nil
}
