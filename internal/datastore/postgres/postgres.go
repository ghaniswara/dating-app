package postgres

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitializeDB initializes the PostgreSQL database connection and returns it
func InitializeDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
		return nil, err // Return nil and the error
	}
	return db, nil // Return the database connection
}
