package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"zelesonic/pilot-ai/types"

	_ "github.com/mattn/go-sqlite3" // The SQLite driver
)

var db *sql.DB

// InitDB initializes the database connection and creates tables if they don't exist.
func InitDB() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get user config dir: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "zelesonic-pilot-ai")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create app config dir: %w", err)
	}
	dbPath := filepath.Join(appConfigDir, "pilot.db")

	// Open the database file, creating it if it doesn't exist.
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	db = database

	log.Println("Database initialized at:", dbPath)
	return createTables()
}

// GetDocumentByID retrieves a single document by its primary key.
func GetDocumentByID(id string) (types.Document, error) {
	var doc types.Document
	var progress sql.NullString
	err := db.QueryRow("SELECT id, file_name, file_path, status, processing_progress FROM documents WHERE id = ?", id).Scan(&doc.ID, &doc.FileName, &doc.FilePath, &doc.Status, &progress)
	if err != nil {
		if err == sql.ErrNoRows {
			return doc, fmt.Errorf("document with ID %s not found", id)
		}
		return doc, err
	}
	if progress.Valid {
		doc.ProcessingProgress = progress.String
	}
	return doc, nil
}


// createTables sets up the database schema.
func createTables() error {
	// SQL statements to create the necessary tables.
	// Using TEXT for JSON blobs (like embeddings and messages) is simple and effective.
	// Using ON DELETE CASCADE simplifies deletion logic.
	sqlStmt := `
    CREATE TABLE IF NOT EXISTS config (
        key TEXT PRIMARY KEY,
        value TEXT
    );
    CREATE TABLE IF NOT EXISTS documents (
        id TEXT PRIMARY KEY,
        file_name TEXT NOT NULL,
        file_path TEXT NOT NULL,
        status TEXT NOT NULL,
        processing_progress TEXT
    );
    CREATE TABLE IF NOT EXISTS chunks (
        chunk_id TEXT PRIMARY KEY,
        document_id TEXT NOT NULL,
        parent_id TEXT,
        type TEXT,
        content TEXT NOT NULL,
        embedding TEXT, -- Stored as a JSON string
        FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE
    );
    `
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	log.Println("Database tables created or verified successfully.")
	return nil
}

// --- Config Functions ---

// SetConfigValue saves or updates a key-value pair in the config table.
func SetConfigValue(key, value string) error {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(key, value)
	return err
}

// GetConfigValue retrieves a value from the config table.
func GetConfigValue(key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil // Return empty string if not found, not an error
	}
	return value, err
}

// --- Document Functions ---

// SaveDocument inserts or updates a document in the database.

func SaveDocument(doc types.Document) error {
	stmt, err := db.Prepare("INSERT OR REPLACE INTO documents (id, file_name, file_path, status) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(doc.ID, doc.FileName, doc.FilePath, doc.Status)
	return err
}

// the GetDocuments function
func GetDocuments() ([]types.Document, error) {
	rows, err := db.Query("SELECT id, file_name, file_path, status, processing_progress FROM documents ORDER BY file_name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []types.Document
	for rows.Next() {
		var doc types.Document
		var progress sql.NullString // Handle potentially null progress field
		if err := rows.Scan(&doc.ID, &doc.FileName, &doc.FilePath, &doc.Status, &progress); err != nil {
			return nil, err
		}
		if progress.Valid {
			doc.ProcessingProgress = progress.String
		}
		docs = append(docs, doc)
	}
	return docs, nil
}


// DeleteDocument removes a document and its associated chunks.
func DeleteDocument(docID string) error {
	// ON DELETE CASCADE in the schema handles deleting chunks automatically.
	stmt, err := db.Prepare("DELETE FROM documents WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(docID)
	return err
}

// UpdateDocumentStatus updates the status of a specific document.
func UpdateDocumentStatus(docID, status string) error {
	stmt, err := db.Prepare("UPDATE documents SET status = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(status, docID)
	return err
}

// --- Chunk Functions ---

func SaveChunk(chunk types.DocumentChunk) error {
    embeddingJSON, err := json.Marshal(chunk.Embedding)
    if err != nil {
        return fmt.Errorf("failed to marshal embedding: %w", err)
    }

    stmt, err := db.Prepare("INSERT INTO chunks (chunk_id, document_id, parent_id, type, content, embedding) VALUES (?, ?, ?, ?, ?, ?)")
    if err != nil {
        return err
    }
    defer stmt.Close()
    _, err = stmt.Exec(chunk.ChunkID, chunk.DocumentID, chunk.ParentID, chunk.Type, chunk.Content, string(embeddingJSON))
    return err
}

// GetAllChunks retrieves all chunks with their embeddings.

func GetAllChunks() ([]types.DocumentChunk, error) {
	rows, err := db.Query("SELECT chunk_id, document_id, parent_id, type, content, embedding FROM chunks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []types.DocumentChunk
	for rows.Next() {
		var chunk types.DocumentChunk
		var embeddingJSON sql.NullString
		var parentID sql.NullString

		if err := rows.Scan(&chunk.ChunkID, &chunk.DocumentID, &parentID, &chunk.Type, &chunk.Content, &embeddingJSON); err != nil {
			return nil, err
		}
		if parentID.Valid {
			chunk.ParentID = parentID.String
		}
		if embeddingJSON.Valid {
			if err := json.Unmarshal([]byte(embeddingJSON.String), &chunk.Embedding); err != nil {
				log.Printf("Warning: failed to unmarshal embedding for chunk %s: %v", chunk.ChunkID, err)
			}
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}
// --- Conversation & Message Functions ---
// ResetAllData clears all user-generated content from the database.
func ResetAllData() error {
    _, err := db.Exec("DELETE FROM chunks; DELETE FROM documents;")
    return err
}

func UpdateDocumentProgress(docID, progress string) error {
	stmt, err := db.Prepare("UPDATE documents SET processing_progress = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(progress, docID)
	return err
}

// Overloaded function to update both status and progress
func UpdateDocumentStatusAndProgress(docID, status, progress string) error {
	stmt, err := db.Prepare("UPDATE documents SET status = ?, processing_progress = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(status, progress, docID)
	return err
}