// main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/mikeshootzz/outline-rag-scraper/docs" // replace with your actual module path after running swag init

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

// ====================================================================
// Swagger General API Info
// ====================================================================
//
// @title Document Export & Upload API
// @version 1.0
// @description This API exports documents from the source API as Markdown files and uploads them to OpenWebUI.
// @host localhost:8080
// @BasePath /
// ====================================================================

// Config holds configuration values read from environment variables.
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
}

// Global configuration variable.
var config Config

// Document represents each document in the response from the docs API.
type Document struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URLId string `json:"urlId"`
}

// DocumentsResponse represents the API response when listing documents.
type DocumentsResponse struct {
	Data []Document `json:"data"`
}

// ExportResponse represents the API response from the document export call.
type ExportResponse struct {
	Data string `json:"data"`
}

// KnowledgeResponse represents the response from OpenWebUI knowledge collection GET.
type KnowledgeResponse struct {
	Files []struct {
		ID string `json:"id"`
	} `json:"files"`
}

// ------------------------
// Helper Functions
// ------------------------

// sanitizeURLTitle converts a title to a URL-friendly string by converting it to lowercase,
// replacing non-alphanumeric characters with hyphens, and trimming leading/trailing hyphens.
func sanitizeURLTitle(title string) string {
	lower := strings.ToLower(title)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	replaced := re.ReplaceAllString(lower, "-")
	return strings.Trim(replaced, "-")
}

// sanitizeFilename creates a safe filename by replacing spaces with underscores and
// removing characters that are not alphanumeric, underscores, or hyphens.
func sanitizeFilename(title string) string {
	title = strings.ReplaceAll(title, " ", "_")
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return re.ReplaceAllString(title, "")
}

// ------------------------
// Export Functions
// ------------------------

// fetchDocuments retrieves a page of documents from the docs API.
func fetchDocuments(offset int) (*DocumentsResponse, error) {
	url := fmt.Sprintf("%s/documents.list", config.APIBaseURL)
	payload := map[string]interface{}{
		"offset":    offset,
		"limit":     config.Limit,
		"sort":      "updatedAt",
		"direction": "DESC",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchDocuments: unexpected status: %s", resp.Status)
	}
	var docsResp DocumentsResponse
	if err = json.NewDecoder(resp.Body).Decode(&docsResp); err != nil {
		return nil, err
	}
	return &docsResp, nil
}

// exportAndSaveDocument calls the export API for a document, prepends the generated URL,
// and saves the result as a Markdown file.
func exportAndSaveDocument(doc Document) error {
	safeURLTitle := sanitizeURLTitle(doc.Title)
	docURL := fmt.Sprintf("%s/%s-%s", config.DocsBaseURL, safeURLTitle, doc.URLId)
	safeTitle := sanitizeFilename(doc.Title)

	url := fmt.Sprintf("%s/documents.export", config.APIBaseURL)
	payload := map[string]interface{}{
		"id": doc.ID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("exportAndSaveDocument: unexpected status: %s", resp.Status)
	}
	var expResp ExportResponse
	if err = json.NewDecoder(resp.Body).Decode(&expResp); err != nil {
		return err
	}
	content := fmt.Sprintf("Document URL: %s\n\n%s", docURL, expResp.Data)
	if err = os.MkdirAll(config.DocumentsDir, os.ModePerm); err != nil {
		return err
	}
	filePath := filepath.Join(config.DocumentsDir, safeTitle+".md")
	if err = ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}
	log.Printf("Downloaded and saved: %s", filePath)
	return nil
}

// exportDocumentsHandler handles the export process.
// @Summary Export documents
// @Description Fetches documents from the source API, exports their content, and saves them as Markdown files.
// @Tags export
// @Produce plain
// @Success 200 {string} string "Export completed."
// @Failure 500 {object} map[string]interface{}
// @Router /export [get]
func exportDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	offset := 0
	for {
		docsResp, err := fetchDocuments(offset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching documents: %v", err), http.StatusInternalServerError)
			return
		}
		if len(docsResp.Data) == 0 {
			break
		}
		for _, doc := range docsResp.Data {
			if err := exportAndSaveDocument(doc); err != nil {
				log.Printf("Error exporting document %s: %v", doc.ID, err)
			}
		}
		offset += config.Limit
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Export completed."))
}

// ------------------------
// Upload Functions
// ------------------------

// clearKnowledgeCollection clears the OpenWebUI knowledge collection by removing all file IDs.
func clearKnowledgeCollection() error {
	url := fmt.Sprintf("%s/knowledge/%s", config.OpenWebUIAPIURL, config.KnowledgeCollectionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.OpenWebUIAPIToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clearKnowledgeCollection: unexpected status: %s", resp.Status)
	}
	var knowResp KnowledgeResponse
	if err = json.NewDecoder(resp.Body).Decode(&knowResp); err != nil {
		return err
	}
	for _, file := range knowResp.Files {
		if err := removeFileFromKnowledge(file.ID); err != nil {
			log.Printf("Error removing file %s: %v", file.ID, err)
		}
	}
	log.Printf("Knowledge collection cleared.")
	return nil
}

// removeFileFromKnowledge removes a single file from the OpenWebUI knowledge collection.
func removeFileFromKnowledge(fileID string) error {
	url := fmt.Sprintf("%s/knowledge/%s/file/remove", config.OpenWebUIAPIURL, config.KnowledgeCollectionID)
	payload := map[string]interface{}{
		"file_id": fileID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.OpenWebUIAPIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("removeFileFromKnowledge: failed with status %s", resp.Status)
	}
	log.Printf("Removed file ID %s from knowledge collection.", fileID)
	return nil
}

// uploadToOpenWebUI uploads a file via multipart form data and adds it to the knowledge collection.
func uploadToOpenWebUI(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err = io.Copy(part, f); err != nil {
		return err
	}
	writer.Close()

	url := fmt.Sprintf("%s/files/", config.OpenWebUIAPIURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.OpenWebUIAPIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("uploadToOpenWebUI: unexpected status: %s, body: %s", resp.Status, string(respBody))
	}
	var uploadResp map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return err
	}
	fileID, ok := uploadResp["id"].(string)
	if !ok || fileID == "" {
		return fmt.Errorf("uploadToOpenWebUI: file ID not found in response")
	}
	log.Printf("Uploaded file %s with ID %s", filePath, fileID)
	return addToKnowledgeCollection(fileID)
}

// addToKnowledgeCollection adds an uploaded file to the OpenWebUI knowledge collection.
func addToKnowledgeCollection(fileID string) error {
	url := fmt.Sprintf("%s/knowledge/%s/file/add", config.OpenWebUIAPIURL, config.KnowledgeCollectionID)
	payload := map[string]interface{}{
		"file_id": fileID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.OpenWebUIAPIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("addToKnowledgeCollection: failed with status %s", resp.Status)
	}
	log.Printf("Added file ID %s to knowledge collection %s", fileID, config.KnowledgeCollectionID)
	return nil
}

// uploadDocumentsHandler handles the upload process.
// @Summary Upload documents
// @Description Clears the OpenWebUI knowledge collection and uploads local Markdown files.
// @Tags upload
// @Produce plain
// @Success 200 {string} string "Upload completed."
// @Failure 500 {object} map[string]interface{}
// @Router /upload [get]
func uploadDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	if err := clearKnowledgeCollection(); err != nil {
		http.Error(w, fmt.Sprintf("Error clearing knowledge collection: %v", err), http.StatusInternalServerError)
		return
	}
	files, err := ioutil.ReadDir(config.DocumentsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading directory: %v", err), http.StatusInternalServerError)
		return
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			filePath := filepath.Join(config.DocumentsDir, file.Name())
			if err := uploadToOpenWebUI(filePath); err != nil {
				log.Printf("Error uploading file %s: %v", filePath, err)
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Upload completed."))
}

// ------------------------
// Main Function
// ------------------------

func main() {
	// Load environment variables from .env file.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env file, continuing with environment variables")
	}

	// Read configuration from environment variables.
	config = Config{
		APIToken:              os.Getenv("API_TOKEN"),
		APIBaseURL:            os.Getenv("API_BASE_URL"),
		DocsBaseURL:           os.Getenv("DOCS_BASE_URL"),
		OpenWebUIAPIToken:     os.Getenv("OPENWEBUI_API_TOKEN"),
		KnowledgeCollectionID: os.Getenv("KNOWLEDGE_COLLECTION_ID"),
		OpenWebUIAPIURL:       os.Getenv("OPENWEBUI_API_URL"),
		DocumentsDir:          os.Getenv("DOCUMENTS_DIR"),
		Port:                  os.Getenv("PORT"),
	}
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.DocumentsDir == "" {
		config.DocumentsDir = "./tmp-files"
	}
	limitStr := os.Getenv("LIMIT")
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			config.Limit = l
		} else {
			config.Limit = 100
		}
	} else {
		config.Limit = 100
	}

	// Optional: Check for required configuration.
	if config.APIBaseURL == "" {
		log.Fatal("API_BASE_URL is not set. Please set it in your .env file.")
	}

	// Set up the router.
	router := mux.NewRouter()

	// Swagger endpoint.
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// API endpoints.
	router.HandleFunc("/export", exportDocumentsHandler).Methods("GET")
	router.HandleFunc("/upload", uploadDocumentsHandler).Methods("GET")

	log.Printf("Server started on :%s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		log.Fatal(err)
	}
}
