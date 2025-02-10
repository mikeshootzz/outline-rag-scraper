package handlers

import "github.com/gorilla/mux"

// RegisterRoutes registers the API endpoints with the router.
func RegisterRoutes(router *mux.Router) {
	// Export endpoint
	router.HandleFunc("/export", ExportDocumentsHandler).Methods("GET")
	// Upload endpoint
	router.HandleFunc("/upload", UploadDocumentsHandler).Methods("GET")
}
