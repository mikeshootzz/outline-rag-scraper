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

	"github.com/mikeshootzz/outline-rag-scraper/config"
	"github.com/mikeshootzz/outline-rag-scraper/models"
	"github.com/mikeshootzz/outline-rag-scraper/utils"
)

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

	resp, err := http.DefaultClient.Do(req)
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

// exportAndSaveDocument exports a single document and saves it as a Markdown file.
func exportAndSaveDocument(doc models.Document) error {
	safeURLTitle := utils.SanitizeURLTitle(doc.Title)
	docURL := fmt.Sprintf("%s/%s-%s", config.ConfigInstance.DocsBaseURL, safeURLTitle, doc.URLId)
	safeTitle := utils.SanitizeFilename(doc.Title)

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

	resp, err := http.DefaultClient.Do(req)
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
	if err = os.MkdirAll(config.ConfigInstance.DocumentsDir, os.ModePerm); err != nil {
		return err
	}
	filePath := filepath.Join(config.ConfigInstance.DocumentsDir, safeTitle+".md")
	if err = ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}
	log.Printf("Downloaded and saved: %s", filePath)
	return nil
}

// ExportDocumentsHandler handles the export process.
// @Summary Export documents
// @Description Fetches documents from the source API, exports their content, and saves them as Markdown files.
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
