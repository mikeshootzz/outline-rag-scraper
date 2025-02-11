package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mikeshootzz/outline-rag-scraper/models"
	"github.com/mikeshootzz/outline-rag-scraper/utils"
)

// MappingPayload represents the expected payload for creating a collection mapping.
type MappingPayload struct {
	OutlineCollection    string   `json:"outline_collection"`    // e.g., "Human_Resources"
	OpenWebUICollections []string `json:"openwebui_collections"` // e.g., ["collectionID1", "collectionID2"]
}

// CreateMappingHandler creates a new collection mapping.
// @Summary Create a new collection mapping
// @Description Creates a mapping between an Outline collection (subdirectory) and one or more OpenWebUI knowledge collections.
// @Tags mappings
// @Accept json
// @Produce json
// @Param mapping body MappingPayload true "Mapping Payload"
// @Success 201 {object} models.CollectionMapping
// @Failure 400 {object} map[string]string "Invalid payload"
// @Failure 500 {object} map[string]string "Failed to create mapping"
// @Router /mappings [post]
func CreateMappingHandler(w http.ResponseWriter, r *http.Request) {
	var payload MappingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	mapping := models.CollectionMapping{
		OutlineCollection:    payload.OutlineCollection,
		OpenWebUICollections: strings.Join(payload.OpenWebUICollections, ","),
	}

	if err := utils.DB.Create(&mapping).Error; err != nil {
		http.Error(w, "Failed to create mapping", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mapping)
}

// GetMappingsHandler retrieves all collection mappings.
// @Summary Get collection mappings
// @Description Retrieves all mappings between Outline collections and OpenWebUI knowledge collections.
// @Tags mappings
// @Produce json
// @Success 200 {array} models.CollectionMapping
// @Failure 500 {object} map[string]string "Failed to retrieve mappings"
// @Router /mappings [get]
func GetMappingsHandler(w http.ResponseWriter, r *http.Request) {
	var mappings []models.CollectionMapping
	if err := utils.DB.Find(&mappings).Error; err != nil {
		http.Error(w, "Failed to retrieve mappings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mappings)
}
