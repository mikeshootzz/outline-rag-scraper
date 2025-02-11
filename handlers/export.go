package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mikeshootzz/outline-rag-scraper/config"
	"github.com/mikeshootzz/outline-rag-scraper/models"
	"github.com/mikeshootzz/outline-rag-scraper/utils"
)

// Global cache for collection names (to avoid repeated API calls)
var (
	collectionCache   = make(map[string]string)
	collectionCacheMu sync.Mutex
)

// doRequestWithRateLimit sends an HTTP request and respects rate limiting.
// If a 429 status code is returned, it reads the "Retry-After" header (which
// specifies the number of milliseconds to wait) before retrying.
func doRequestWithRateLimit(req *http.Request) (*http.Response, error) {
	for {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		// If we are not rate-limited, return the response.
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Otherwise, read the Retry-After header.
		retryAfterStr := resp.Header.Get("Retry-After")
		var waitDuration time.Duration
		if retryAfterStr != "" {
			ms, err := strconv.Atoi(retryAfterStr)
			if err != nil {
				waitDuration = 1 * time.Second
			} else {
				waitDuration = time.Duration(ms) * time.Millisecond
			}
		} else {
			waitDuration = 1 * time.Second
		}
		log.Printf("Rate limited: waiting for %v before retrying...", waitDuration)
		resp.Body.Close() // Make sure to close the response body before sleeping.
		time.Sleep(waitDuration)
	}
}

// fetchDocuments retrieves a page of documents from the docs API.
func fetchDocuments(offset int) (*models.DocumentsResponse, error) {
	url := fmt.Sprintf("%s/documents.list", config.ConfigInstance.APIBaseURL)
	payload := map[string]interface{}{
		"offset":    offset,
		"limit":     config.ConfigInstance.Limit,
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
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := doRequestWithRateLimit(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchDocuments: unexpected status: %s", resp.Status)
	}
	var docsResp models.DocumentsResponse
	if err = json.NewDecoder(resp.Body).Decode(&docsResp); err != nil {
		return nil, err
	}
	return &docsResp, nil
}

// fetchCollectionName retrieves the collection name for a given collectionID.
// It uses caching to avoid duplicate API calls.
func fetchCollectionName(collectionID string) (string, error) {
	// Check if the collection name is already in the cache.
	collectionCacheMu.Lock()
	if name, exists := collectionCache[collectionID]; exists {
		collectionCacheMu.Unlock()
		return name, nil
	}
	collectionCacheMu.Unlock()

	// Make API call to fetch the collection info.
	url := fmt.Sprintf("%s/collections.info", config.ConfigInstance.APIBaseURL)
	payload := map[string]interface{}{
		"id": collectionID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := doRequestWithRateLimit(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetchCollectionName: unexpected status: %s", resp.Status)
	}

	// Define a temporary struct to parse the response.
	var collResp struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&collResp); err != nil {
		return "", err
	}

	// Cache the collection name for future lookups.
	collectionCacheMu.Lock()
	collectionCache[collectionID] = collResp.Data.Name
	collectionCacheMu.Unlock()

	return collResp.Data.Name, nil
}

// exportAndSaveDocument exports a single document and saves it as a Markdown file,
// grouping it into a subdirectory based on its collection.
func exportAndSaveDocument(doc models.Document) error {
	// Create a URL-safe and file-safe title for the document.
	safeURLTitle := utils.SanitizeURLTitle(doc.Title)
	docURL := fmt.Sprintf("%s/%s-%s", config.ConfigInstance.DocsBaseURL, safeURLTitle, doc.URLId)
	safeTitle := utils.SanitizeFilename(doc.Title)

	// Export the document using the API.
	url := fmt.Sprintf("%s/documents.export", config.ConfigInstance.APIBaseURL)
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
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := doRequestWithRateLimit(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("exportAndSaveDocument: unexpected status: %s", resp.Status)
	}
	var expResp models.ExportResponse
	if err = json.NewDecoder(resp.Body).Decode(&expResp); err != nil {
		return err
	}
	content := fmt.Sprintf("Document URL: %s\n\n%s", docURL, expResp.Data)

	// Determine the directory path based on the document's collection.
	var dirPath string
	if doc.CollectionId != "" {
		collectionName, err := fetchCollectionName(doc.CollectionId)
		if err != nil {
			log.Printf("Error fetching collection name for document %s: %v", doc.ID, err)
			// If the collection lookup fails, use the base documents directory.
			dirPath = config.ConfigInstance.DocumentsDir
		} else {
			// Sanitize the collection name to be safe for a directory name.
			safeCollectionName := utils.SanitizeFilename(collectionName)
			dirPath = filepath.Join(config.ConfigInstance.DocumentsDir, safeCollectionName)
		}
	} else {
		// If no collection ID is provided, fall back to the base directory.
		dirPath = config.ConfigInstance.DocumentsDir
	}

	// Ensure the directory exists.
	if err = os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}
	// Create the file path within the subdirectory.
	filePath := filepath.Join(dirPath, safeTitle+".md")
	if err = ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}
	log.Printf("Downloaded and saved: %s", filePath)
	return nil
}

// ExportDocumentsHandler handles the export process.
// @Summary Export documents
// @Description Fetches documents from the source API, exports their content, and saves them as Markdown files grouped by collection.
// @Tags export
// @Produce plain
// @Success 200 {string} string "Export completed."
// @Failure 500 {object} map[string]interface{}
// @Router /export [get]
func ExportDocumentsHandler(w http.ResponseWriter, r *http.Request) {
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
		offset += config.ConfigInstance.Limit
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Export completed."))
}
