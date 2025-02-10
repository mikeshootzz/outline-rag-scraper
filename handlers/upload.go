package handlers

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
	"strings"

	"github.com/mikeshootzz/outline-rag-scraper/config"
	"github.com/mikeshootzz/outline-rag-scraper/models"
)

// clearKnowledgeCollection clears the OpenWebUI knowledge collection.
func clearKnowledgeCollection() error {
	url := fmt.Sprintf("%s/knowledge/%s", config.ConfigInstance.OpenWebUIAPIURL, config.ConfigInstance.KnowledgeCollectionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.OpenWebUIAPIToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clearKnowledgeCollection: unexpected status: %s", resp.Status)
	}
	var knowResp models.KnowledgeResponse
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

// removeFileFromKnowledge removes a file from the OpenWebUI knowledge collection.
func removeFileFromKnowledge(fileID string) error {
	url := fmt.Sprintf("%s/knowledge/%s/file/remove", config.ConfigInstance.OpenWebUIAPIURL, config.ConfigInstance.KnowledgeCollectionID)
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
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.OpenWebUIAPIToken)
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

// uploadToOpenWebUI uploads a file via multipart form data.
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

	url := fmt.Sprintf("%s/files/", config.ConfigInstance.OpenWebUIAPIURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.OpenWebUIAPIToken)
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

// addToKnowledgeCollection adds an uploaded file to the knowledge collection.
func addToKnowledgeCollection(fileID string) error {
	url := fmt.Sprintf("%s/knowledge/%s/file/add", config.ConfigInstance.OpenWebUIAPIURL, config.ConfigInstance.KnowledgeCollectionID)
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
	req.Header.Set("Authorization", "Bearer "+config.ConfigInstance.OpenWebUIAPIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("addToKnowledgeCollection: failed with status %s", resp.Status)
	}
	log.Printf("Added file ID %s to knowledge collection %s", fileID, config.ConfigInstance.KnowledgeCollectionID)
	return nil
}

// UploadDocumentsHandler handles the upload process.
// @Summary Upload documents
// @Description Clears the OpenWebUI knowledge collection and uploads local Markdown files.
// @Tags upload
// @Produce plain
// @Success 200 {string} string "Upload completed."
// @Failure 500 {object} map[string]interface{}
// @Router /upload [get]
func UploadDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	if err := clearKnowledgeCollection(); err != nil {
		http.Error(w, fmt.Sprintf("Error clearing knowledge collection: %v", err), http.StatusInternalServerError)
		return
	}
	files, err := ioutil.ReadDir(config.ConfigInstance.DocumentsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading directory: %v", err), http.StatusInternalServerError)
		return
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			filePath := filepath.Join(config.ConfigInstance.DocumentsDir, file.Name())
			if err := uploadToOpenWebUI(filePath); err != nil {
				log.Printf("Error uploading file %s: %v", filePath, err)
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Upload completed."))
}
