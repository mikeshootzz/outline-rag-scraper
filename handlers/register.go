// handlers/register.go
package handlers

import "github.com/gorilla/mux"

// RegisterRoutes registers the API endpoints with the router.
func RegisterRoutes(router *mux.Router) {
	// Export endpoint
	router.HandleFunc("/export", ExportDocumentsHandler).Methods("GET")
	// Upload endpoint
	router.HandleFunc("/upload", UploadDocumentsHandler).Methods("GET")
	// Mapping endpoints
	router.HandleFunc("/mappings", CreateMappingHandler).Methods("POST")
	router.HandleFunc("/mappings", GetMappingsHandler).Methods("GET")
}
