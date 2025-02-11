// utils/db.go
package utils

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mikeshootzz/outline-rag-scraper/config"
	"github.com/mikeshootzz/outline-rag-scraper/models"
)

// DB is the global database connection.
var DB *gorm.DB

// InitDB initializes the PostgreSQL database connection using GORM.
func InitDB() {
	dsn := config.ConfigInstance.DatabaseURL // e.g.: "host=localhost user=youruser password=yourpassword dbname=yourdb port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	DB = db

	// Automatically migrate the CollectionMapping model.
	if err := db.AutoMigrate(&models.CollectionMapping{}); err != nil {
		log.Fatalf("failed to auto-migrate database: %v", err)
	}

	log.Println("Database connection initialized.")
}
