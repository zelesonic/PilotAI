// types/types.go
package types

// Document holds metadata for an uploaded file.
type Document struct {
    ID                 string `json:"id"`
    FileName           string `json:"fileName"`
    FilePath           string `json:"filePath"`
    Status             string `json:"status"`             // Can be "processing", "completed", "failed"
    ProcessingProgress string `json:"processingProgress"` // To hold messages like "Embedding chunk 1/500"
}

// DocumentChunk is the core data structure for a piece of processed text.
type DocumentChunk struct {
    ChunkID    string    `json:"chunkId"`
    ParentID   string    `json:"parentId"`
    Type       string    `json:"type"` // "summary" or "detail"
    Content    string    `json:"content"`
    Embedding  []float64 `json:"embedding"`
    DocumentID string    `json:"documentId"`
}