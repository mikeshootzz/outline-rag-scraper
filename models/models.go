package models

// Document represents a single document.
type Document struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URLId string `json:"urlId"`
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
