package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds configuration values.
type Config struct {
	APIToken              string
	APIBaseURL            string
	DocsBaseURL           string
	OpenWebUIAPIToken     string
	KnowledgeCollectionID string
	OpenWebUIAPIURL       string
	DocumentsDir          string
	Limit                 int
	Port                  string
	DatabaseURL           string // New field for your PostgreSQL DSN.
}

// ConfigInstance is the global configuration instance.
var ConfigInstance Config

// LoadConfig reads environment variables and initializes ConfigInstance.
func LoadConfig() {
	ConfigInstance = Config{
		APIToken:              os.Getenv("API_TOKEN"),
		APIBaseURL:            os.Getenv("API_BASE_URL"),
		DocsBaseURL:           os.Getenv("DOCS_BASE_URL"),
		OpenWebUIAPIToken:     os.Getenv("OPENWEBUI_API_TOKEN"),
		KnowledgeCollectionID: os.Getenv("KNOWLEDGE_COLLECTION_ID"),
		OpenWebUIAPIURL:       os.Getenv("OPENWEBUI_API_URL"),
		DocumentsDir:          os.Getenv("DOCUMENTS_DIR"),
		Port:                  os.Getenv("PORT"),
		DatabaseURL:           os.Getenv("DATABASE_URL"), // Load the database URL from your env.
	}

	if ConfigInstance.Port == "" {
		ConfigInstance.Port = "8080"
	}
	if ConfigInstance.DocumentsDir == "" {
		ConfigInstance.DocumentsDir = "./tmp-files"
	}
	limitStr := os.Getenv("LIMIT")
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			ConfigInstance.Limit = l
		} else {
			ConfigInstance.Limit = 100
		}
	} else {
		ConfigInstance.Limit = 100
	}

	// Optional: Ensure required values are set.
	if ConfigInstance.APIBaseURL == "" {
		log.Fatal("API_BASE_URL is not set. Please set it in your .env file.")
	}
}
