package main

import (
	"log"
	"net/http"

	_ "github.com/mikeshootzz/outline-rag-scraper/docs" // Replace with your actual module path

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/mikeshootzz/outline-rag-scraper/config"
	"github.com/mikeshootzz/outline-rag-scraper/handlers"
	"github.com/mikeshootzz/outline-rag-scraper/utils" // Import the utils package for DB initialization.
)

func main() {
	// Load environment variables from .env file.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env file, continuing with environment variables")
	}

	// Load configuration (populates config.ConfigInstance).
	config.LoadConfig()

	// Initialize the PostgreSQL database connection.
	utils.InitDB()

	// Create a new router.
	router := mux.NewRouter()

	// Register API endpoints.
	handlers.RegisterRoutes(router)

	// Serve Swagger UI at /docs (e.g., http://localhost:8080/docs/index.html)
	router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)

	log.Printf("Server started on :%s", config.ConfigInstance.Port)
	if err := http.ListenAndServe(":"+config.ConfigInstance.Port, router); err != nil {
		log.Fatal(err)
	}
}
