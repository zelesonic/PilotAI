package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"zelesonic/pilot-ai/database"
	"zelesonic/pilot-ai/index"
	"zelesonic/pilot-ai/processors"
	"zelesonic/pilot-ai/types"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

//go:embed all:frontend
var frontendFS embed.FS
var memoryIndex *index.MemoryIndex

// --- Main Application Setup ---

func main() {
	// Initialize the database
	if err := database.InitDB(); err != nil {
		log.Fatalf("Fatal Error: Could not initialize database: %v", err)
	}

	// --- Create and Populate the In-Memory Index on Startup ---
	log.Println("Initializing in-memory vector index...")
	memoryIndex = index.New()
	existingChunks, err := database.GetAllChunks()
	if err != nil {
		log.Fatalf("Fatal Error: Could not load existing chunks for indexing: %v", err)
	}
	for _, chunk := range existingChunks {
		if len(chunk.Embedding) > 0 {
			memoryIndex.Add(chunk)
		}
	}
	log.Printf("In-memory index created with %d vectors.", len(existingChunks))
	// --- End of Indexing ---

	port := "5000"
	serverURL := "http://localhost:" + port

	mux := http.NewServeMux()

	// --- Static File Handler ---
	contentFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(contentFS)))

	// --- API Handlers (Authentication Removed) ---
	mux.HandleFunc("/api/ollama/status", corsMiddleware(http.HandlerFunc(ollamaStatusHandler)).ServeHTTP)
	mux.HandleFunc("/api/ollama/config", corsMiddleware(http.HandlerFunc(ollamaConfigHandler)).ServeHTTP)
	mux.HandleFunc("/api/ollama/models/local", corsMiddleware(http.HandlerFunc(localModelsHandler)).ServeHTTP)
	mux.HandleFunc("/api/ollama/models/delete", corsMiddleware(http.HandlerFunc(deleteModelHandler)).ServeHTTP)
	mux.HandleFunc("/api/ollama/active_models", corsMiddleware(http.HandlerFunc(activeModelsHandler)).ServeHTTP)

	// Endpoints are now public for the open-source version.
	mux.HandleFunc("/api/upload", corsMiddleware(http.HandlerFunc(uploadHandler)).ServeHTTP)
	mux.HandleFunc("/api/chat", corsMiddleware(http.HandlerFunc(chatHandler)).ServeHTTP)
	mux.HandleFunc("/api/documents", corsMiddleware(http.HandlerFunc(documentsHandler)).ServeHTTP)
	mux.HandleFunc("/api/documents/delete", corsMiddleware(http.HandlerFunc(deleteDocumentHandler)).ServeHTTP)
	mux.HandleFunc("/api/reset", corsMiddleware(http.HandlerFunc(resetHandler)).ServeHTTP)
	mux.HandleFunc("/api/documents/select", corsMiddleware(http.HandlerFunc(selectDocumentHandler)).ServeHTTP)
	mux.HandleFunc("/api/execute", corsMiddleware(http.HandlerFunc(executeHandler)).ServeHTTP)

	// --- Server Startup Logic ---
	log.Printf("Starting Zelesonic Pilot AI server on %s...", serverURL)
	log.Println("Open your web browser and navigate to http://localhost:5000 to use the application.")
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// --- Refactored Handlers ---

func selectDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	if err := database.SetConfigValue("activeDocumentID", reqBody.ID); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set active document"})
		return
	}
	log.Printf("Active document set to: %s", reqBody.ID)
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func ollamaStatusHandler(w http.ResponseWriter, r *http.Request) {
	baseURL, _ := database.GetConfigValue("ollamaBaseURL")
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default value
	}
	client, err := createOllamaClient(baseURL)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.List(ctx)

	activeEmbedding, _ := database.GetConfigValue("activeEmbeddingModel")
	activeGenerative, _ := database.GetConfigValue("activeGenerativeModel")

	statusPayload := map[string]interface{}{
		"active_embedding_model":  activeEmbedding,
		"active_generative_model": activeGenerative,
		"current_base_url":        baseURL,
	}
	if err != nil {
		statusPayload["message"] = "Ollama server is not reachable. Please ensure it's running."
	} else {
		statusPayload["message"] = "Ollama is running."
	}
	respondWithJSON(w, http.StatusOK, statusPayload)
}

func ollamaConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		embeddingJSON, _ := database.GetConfigValue("savedEmbeddingModels")
		generativeJSON, _ := database.GetConfigValue("savedGenerativeModels")
		var embeddingModels, generativeModels []string
		json.Unmarshal([]byte(embeddingJSON), &embeddingModels)
		json.Unmarshal([]byte(generativeJSON), &generativeModels)
		payload := map[string]map[string][]string{"saved_models": {"embedding": embeddingModels, "generative": generativeModels}}
		respondWithJSON(w, http.StatusOK, payload)
		return
	}

	if r.Method == http.MethodPost {
		var reqBody struct {
			OllamaBaseURL       string `json:"ollama_base_url"`
			EmbeddingModelName  string `json:"embedding_model_name"`
			GenerativeModelName string `json:"generative_model_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
			return
		}

		if reqBody.OllamaBaseURL != "" {
			database.SetConfigValue("ollamaBaseURL", reqBody.OllamaBaseURL)
		}

		if reqBody.EmbeddingModelName != "" {
			embeddingJSON, _ := database.GetConfigValue("savedEmbeddingModels")
			var models []string
			json.Unmarshal([]byte(embeddingJSON), &models)
			models = appendIfMissing(models, reqBody.EmbeddingModelName)
			newJSON, _ := json.Marshal(models)
			database.SetConfigValue("savedEmbeddingModels", string(newJSON))
		}

		if reqBody.GenerativeModelName != "" {
			generativeJSON, _ := database.GetConfigValue("savedGenerativeModels")
			var models []string
			json.Unmarshal([]byte(generativeJSON), &models)
			models = appendIfMissing(models, reqBody.GenerativeModelName)
			newJSON, _ := json.Marshal(models)
			database.SetConfigValue("savedGenerativeModels", string(newJSON))
		}

		respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Configuration saved successfully."})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func localModelsHandler(w http.ResponseWriter, r *http.Request) {
	baseURL, _ := database.GetConfigValue("ollamaBaseURL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	client, err := createOllamaClient(baseURL)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	list, err := client.List(context.Background())
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not fetch models from Ollama"})
		return
	}
	var modelNames []string
	for _, model := range list.Models {
		modelNames = append(modelNames, model.Name)
	}
	respondWithJSON(w, http.StatusOK, map[string][]string{"models": modelNames})
}

func deleteModelHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		ModelName string `json:"model_name"`
		ModelType string `json:"model_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	configKey := "saved" + strings.Title(reqBody.ModelType) + "Models"
	jsonStr, _ := database.GetConfigValue(configKey)
	var models []string
	json.Unmarshal([]byte(jsonStr), &models)

	var newModels []string
	for _, m := range models {
		if m != reqBody.ModelName {
			newModels = append(newModels, m)
		}
	}

	newJSON, _ := json.Marshal(newModels)
	database.SetConfigValue(configKey, string(newJSON))

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Model deleted successfully."})
}

func activeModelsHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		ActiveEmbeddingModel  string `json:"active_embedding_model"`
		ActiveGenerativeModel string `json:"active_generative_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	database.SetConfigValue("activeEmbeddingModel", reqBody.ActiveEmbeddingModel)
	database.SetConfigValue("activeGenerativeModel", reqBody.ActiveGenerativeModel)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":                  "success",
		"message":                 "Models activated successfully.",
		"active_embedding_model":  reqBody.ActiveEmbeddingModel,
		"active_generative_model": reqBody.ActiveGenerativeModel,
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	activeEmbeddingModel, _ := database.GetConfigValue("activeEmbeddingModel")
	if activeEmbeddingModel == "" {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Please activate an embedding model first."})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "File is too large."})
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid file upload request."})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to read file."})
		return
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get user config dir."})
		return
	}
	uploadsDir := filepath.Join(configDir, "zelesonic-pilot-ai", "uploads")
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create uploads directory."})
		return
	}

	persistentFilePath := filepath.Join(uploadsDir, handler.Filename)
	if err := os.WriteFile(persistentFilePath, fileBytes, 0o666); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save file."})
		return
	}

	newDocID := uuid.New().String()
	newDoc := types.Document{
		ID:                 newDocID,
		FileName:           handler.Filename,
		FilePath:           persistentFilePath,
		Status:             "processing",
		ProcessingProgress: "Starting...",
	}
	if err := database.SaveDocument(newDoc); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save document record."})
		return
	}

	go func() {
		log.Printf("Starting background processing for document ID: %s", newDocID)
		ctx := context.Background()

		fileExtension := strings.ToLower(filepath.Ext(handler.Filename))
		processor, err := processors.NewProcessorForFile(fileExtension)
		if err != nil {
			database.UpdateDocumentStatusAndProgress(newDocID, "failed", "Unsupported file type")
			return
		}

		chunks, err := processor.Process(persistentFilePath, handler.Filename, newDocID)
		if err != nil {
			database.UpdateDocumentStatusAndProgress(newDocID, "failed", "Failed to process document")
			return
		}

		baseURL, _ := database.GetConfigValue("ollamaBaseURL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		ollamaClient, err := createOllamaClient(baseURL)
		if err != nil {
			database.UpdateDocumentStatusAndProgress(newDocID, "failed", "Could not create Ollama client")
			return
		}

		for i, chunk := range chunks {
			progressMsg := fmt.Sprintf("Embedding chunk %d/%d...", i+1, len(chunks))
			log.Println(progressMsg)
			database.UpdateDocumentProgress(newDocID, progressMsg)

			req := &api.EmbeddingRequest{Model: activeEmbeddingModel, Prompt: chunk.Content}
			resp, err := ollamaClient.Embeddings(ctx, req)
			if err != nil {
				log.Printf("Error embedding chunk %d: %v", i+1, err)
				database.UpdateDocumentStatusAndProgress(newDocID, "failed", "Failed to create embeddings")
				return
			}
			chunk.Embedding = resp.Embedding
			if err := database.SaveChunk(chunk); err != nil {
				log.Printf("Error saving chunk %d: %v", i+1, err)
				database.UpdateDocumentStatusAndProgress(newDocID, "failed", "Failed to save embeddings")
				return
			}
			memoryIndex.Add(chunk)
		}

		database.UpdateDocumentStatusAndProgress(newDocID, "completed", "Processing complete")
		log.Printf("Successfully stored embeddings for '%s'.", handler.Filename)
	}()

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status":     "processing_started",
		"documentId": newDocID,
	})
}
func chatHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received. Generating code...")

	activeGenerativeModel, _ := database.GetConfigValue("activeGenerativeModel")
	activeDocumentID, _ := database.GetConfigValue("activeDocumentID")

	var reqBody struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	var schema []string
	if activeDocumentID != "" {
		doc, err := database.GetDocumentByID(activeDocumentID)
		if err == nil {
			schema, _ = processors.GetSchema(doc.FilePath)
		}
	}

	ollamaClient, err := createOllamaClient("http://localhost:11434") // Assuming default URL
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not create Ollama client."})
		return
	}

	codeGenPrompt := fmt.Sprintf(`You are an expert Python data analyst. Your goal is to write a complete, self-contained Python script to answer the user's question.

**Instructions:**
1. A pandas DataFrame named 'df' is already loaded with the user's data. You must use it.
2. The data has the following columns, if available: %s
3. **LOGIC:** Pay close attention to the user's exact words. If they ask for 'Payment Method', use the 'Payment Method' column.
4. **PANDAS SYNTAX (CRITICAL):**
   - When aggregating, the function for counting is 'count' (lowercase c). Do not use 'Count'.
   - When searching text with .str.contains(), always include na=False.
5. **TEXT OUTPUT:** To display any text, data, or summaries, you MUST use the print() function.
6. **CHARTING:** If the user asks for a plot, you MUST use 'matplotlib.pyplot'. DO NOT call plt.show(). You MUST save the figure to the path from sys.argv[1]. Use this exact line: plt.savefig(sys.argv[1], dpi=300, bbox_inches='tight').

User Question: "%s"

Python Code:`, strings.Join(schema, ", "), reqBody.Prompt)

	var pythonCode string
	genErr := ollamaClient.Generate(r.Context(), &api.GenerateRequest{Model: activeGenerativeModel, Prompt: codeGenPrompt, Stream: new(bool)}, func(resp api.GenerateResponse) error {
		pythonCode += resp.Response
		return nil
	})

	if genErr != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "AI failed to generate code."})
		return
	}

	re := regexp.MustCompile("(?s)```python\n(.*?)\n```")
	matches := re.FindStringSubmatch(pythonCode)
	var cleanCode string
	if len(matches) >= 2 {
		cleanCode = strings.TrimSpace(matches[1])
	} else {
		cleanCode = strings.TrimSpace(pythonCode)
	}

	originalLines := strings.Split(cleanCode, "\n")
	var sanitizedLines []string
	for _, line := range originalLines {
		if !strings.Contains(line, "pd.read_csv") && !strings.Contains(line, "pd.read_excel") {
			sanitizedLines = append(sanitizedLines, line)
		}
	}
	finalCode := strings.Join(sanitizedLines, "\n")

	respondWithJSON(w, http.StatusOK, map[string]string{"code": finalCode})

}

func executeHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body for execution"})
		return
	}

	activeDocumentID, _ := database.GetConfigValue("activeDocumentID")
	if activeDocumentID == "" {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "No document has been selected for analysis."})
		return
	}

	doc, err := database.GetDocumentByID(activeDocumentID)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve the selected document."})
		return
	}

	var loaderLine string
	fileExtension := filepath.Ext(doc.FilePath)
	if fileExtension == ".csv" {
		loaderLine = fmt.Sprintf("df = pd.read_csv(r'%s')", doc.FilePath)
	} else {
		loaderLine = fmt.Sprintf("df = pd.read_excel(r'%s')", doc.FilePath)
	}

	preamble := `import pandas as pd
import sys
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

pd.set_option('display.max_rows', None)
pd.set_option('display.max_columns', None)
pd.set_option('display.width', 1000)`

	fullCode := fmt.Sprintf("%s\n\n%s\n\n%s", preamble, loaderLine, reqBody.Code)

	chartFileName := fmt.Sprintf("%s.png", uuid.New().String())
	chartPath := filepath.Join(os.TempDir(), chartFileName)
	defer os.Remove(chartPath)

	log.Println("Executing user-provided code...")
	result, err := executePythonCode(fullCode, chartPath)
	if err != nil {
		log.Printf("Execution failed: %v", err)
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var base64Chart string
	if fileInfo, err := os.Stat(chartPath); err == nil {
		if fileInfo.Size() > 1024 { // 1KB threshold
			chartBytes, err := os.ReadFile(chartPath)
			if err == nil {
				base64Chart = "data:image/png;base64," + base64.StdEncoding.EncodeToString(chartBytes)
			}
		} else {
			log.Println("Skipping empty plot file.")
		}
	}

	log.Println("Execution successful.")
	respondWithJSON(w, http.StatusOK, map[string]string{"result": result, "chart": base64Chart})
}

func executePythonCode(code, chartPath string) (string, error) {
	log.Printf("Attempting to execute Python script...")
	tmpfile, err := os.CreateTemp("", "zelesonic-pilot-ai-*.py")
	if err != nil {
		log.Printf("Failed to create temp python script: %v", err)
		return "", fmt.Errorf("failed to create temp python script: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		log.Printf("Failed to write code to temp python script %s: %v", tmpfile.Name(), err)
		return "", fmt.Errorf("failed to write to temp python script: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Printf("Failed to close temp python script %s: %v", tmpfile.Name(), err)
		return "", fmt.Errorf("failed to close temp python script: %w", err)
	}
	log.Printf("Python script written to: %s", tmpfile.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", tmpfile.Name(), chartPath)
	log.Printf("Executing Python command: %s %s %s", cmd.Path, tmpfile.Name(), chartPath)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Python script execution failed. Error: %v", err)
		log.Printf("Python stdout: \n%s\n", out.String())
		log.Printf("Python stderr: \n%s\n", stderr.String())
		return "", fmt.Errorf("python script execution failed: %s (stdout: %s, stderr: %s)", err.Error(), out.String(), stderr.String())
	}

	log.Println("Python script executed successfully.")
	return out.String(), nil
}

func documentsHandler(w http.ResponseWriter, r *http.Request) {
	docs, err := database.GetDocuments()
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve documents"})
		return
	}
	if docs == nil {
		docs = []types.Document{} // Ensure we return an empty array, not null
	}

	activeDocumentID, _ := database.GetConfigValue("activeDocumentID")

	type responsePayload struct {
		Documents        []types.Document `json:"documents"`
		ActiveDocumentID string           `json:"activeDocumentID"`
	}

	payload := responsePayload{
		Documents:        docs,
		ActiveDocumentID: activeDocumentID,
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func deleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	if err := database.DeleteDocument(reqBody.ID); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete document"})
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if err := database.ResetAllData(); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to reset data"})
		return
	}
	database.SetConfigValue("activeConversationId", "")
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}

func createOllamaClient(baseURL string) (*api.Client, error) {
	ollamaURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama base URL: %w", err)
	}
	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
	}
	return api.NewClient(ollamaURL, httpClient), nil
}

func appendIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}