package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/musiermoore/oksana-vpn-api/internal/database"
	"github.com/musiermoore/oksana-vpn-api/internal/router"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system environment variables")
	}

	db, dbError := initDatabase()

	if dbError != nil {
		return dbError
	}

	// Close the database when the application stops.
	defer db.Close()

	log.Println("MySQL connected successfully")

	return router.StartRouter(db)
}

func initDatabase() (*sql.DB, error) {
	dsn, dsnErr := database.BuildDSN(
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	)

	log.Println(dsn)
	if dsnErr != nil {
		return nil, dsnErr
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	db, err := database.NewMySQL(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}
