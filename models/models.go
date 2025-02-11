// models/models.go
package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Document represents a single document.
type Document struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	URLId        string `json:"urlId"`
	CollectionId string `json:"collectionId"` // Added to track Outline collection ID
}

// DocumentsResponse represents the API response when listing documents.
type DocumentsResponse struct {
	Data []Document `json:"data"`
}

// ExportResponse represents the API response from the export endpoint.
type ExportResponse struct {
	Data string `json:"data"`
}

// KnowledgeResponse represents the response from the OpenWebUI knowledge collection GET.
type KnowledgeResponse struct {
	Files []struct {
		ID string `json:"id"`
	} `json:"files"`
}

// CollectionMapping maps an Outline collection (identified by its sanitized name)
// to one or more OpenWebUI knowledge collection IDs (stored as a comma-separated string).
type CollectionMapping struct {
	// ID is the primary key.
	ID uint `gorm:"primaryKey" json:"id" example:"1"`
	// CreatedAt is a timestamp for when the record was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is a timestamp for when the record was last updated.
	UpdatedAt time.Time `json:"updated_at"`
	// DeletedAt is a timestamp for when the record was deleted.
	// The swaggerignore tag tells swag to skip this field in the documentation.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggerignore:"true"`

	// OutlineCollection should match the sanitized subdirectory name.
	OutlineCollection string `gorm:"uniqueIndex;not null" json:"outline_collection" example:"Human_Resources"`
	// OpenWebUICollections is a comma-separated list of OpenWebUI knowledge collection IDs.
	OpenWebUICollections string `gorm:"not null" json:"openwebui_collections" example:"collectionID1,collectionID2"`
}

// GetCollectionMappings returns a map where the key is the Outline collection (subdirectory)
// and the value is a slice of OpenWebUI knowledge collection IDs.
func GetCollectionMappings(db *gorm.DB) (map[string][]string, error) {
	var mappings []CollectionMapping
	if err := db.Find(&mappings).Error; err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, mapping := range mappings {
		// Split the comma-separated list and trim any extra spaces.
		ids := strings.Split(mapping.OpenWebUICollections, ",")
		for i := range ids {
			ids[i] = strings.TrimSpace(ids[i])
		}
		result[mapping.OutlineCollection] = ids
	}
	return result, nil
}
