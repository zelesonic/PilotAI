package processors

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"zelesonic/pilot-ai/types"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)
// FileProcessor defines the interface for processing different file types.
type FileProcessor interface {
	Process(filePath, originalFileName, documentID string) ([]types.DocumentChunk, error)
}

// TextProcessor handles text-based documents.
type TextProcessor struct{}

// TabularProcessor handles CSV and XLSX files.
type TabularProcessor struct{}

// NewProcessorForFile is a factory function that returns the correct processor for a given file extension.
func NewProcessorForFile(extension string) (FileProcessor, error) {
	switch extension {
	// NOTE: We are removing the unsupported text formats for now to focus on tabular data.
	// We can add them back later with robust processing.
	case ".csv", ".xlsx":
		return &TabularProcessor{}, nil
	default:
		// Changed to only support tabular for now to ensure quality.
		return nil, fmt.Errorf("unsupported file type: %s. Only CSV and XLSX are currently supported", extension)
	}
}

// --- TabularProcessor (Restored and Improved Logic) ---

func (p *TabularProcessor) Process(filePath, originalFileName, documentID string) ([]types.DocumentChunk, error) {
	fileExtension := strings.ToLower(filepath.Ext(originalFileName))
	var allChunks []types.DocumentChunk

	var sheets map[string][][]string
	var err error

	switch fileExtension {
	case ".csv":
		sheets, err = readCsvFile(filePath)
	case ".xlsx":
		sheets, err = readXlsxFile(filePath)
	default:
		return nil, fmt.Errorf("unsupported tabular file type: %s", fileExtension)
	}

	if err != nil {
		return nil, err
	}

	for sheetName, records := range sheets {
		if len(records) < 2 { // Must have at least a header and one data row
			continue
		}
		headers := records[0]

		// Create a single, high-level summary chunk for the entire sheet.
		summaryID := uuid.New().String()
		summaryContent := fmt.Sprintf("The file '%s' contains a sheet named '%s' with the columns: %s.",
			originalFileName, sheetName, strings.Join(headers, ", "))
		allChunks = append(allChunks, types.DocumentChunk{
			ChunkID:    summaryID,
			Type:       "summary",
			Content:    summaryContent,
			DocumentID: documentID,
		})

		// ** THE CRITICAL FIX IS HERE **
		// This creates a dense, fact-based chunk for each row, which is much better for RAG.
		for _, row := range records[1:] { // Start from the first data row
			var builder strings.Builder
			// Prepending context about the source helps the AI.
			builder.WriteString(fmt.Sprintf("From sheet '%s' in file '%s', one record shows: ", sheetName, originalFileName))
			for i, cell := range row {
				if i < len(headers) && strings.TrimSpace(cell) != "" {
					// Format: "header: value; " - This is direct and effective.
					builder.WriteString(fmt.Sprintf("%s: %s; ", strings.TrimSpace(headers[i]), strings.TrimSpace(cell)))
				}
			}
			// Each row becomes a distinct "detail" chunk.
			allChunks = append(allChunks, types.DocumentChunk{
				ChunkID:    uuid.New().String(),
				ParentID:   summaryID,
				Type:       "detail",
				Content:    strings.TrimSpace(builder.String()),
				DocumentID: documentID,
			})
		}
	}
	return allChunks, nil
}

// --- Helper functions for reading tabular data ---

func readCsvFile(filePath string) (map[string][][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	sheetData := make(map[string][][]string)
	// CSV files don't have sheet names, so we use a default.
	sheetData["DefaultSheet"] = records
	return sheetData, nil
}

func readXlsxFile(filePath string) (map[string][][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheetData := make(map[string][][]string)
	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			// Log the error but continue to other sheets if possible
			log.Printf("Warning: Could not read rows from sheet '%s': %v", sheetName, err)
			continue
		}
		sheetData[sheetName] = rows
	}
	return sheetData, nil
}

// Add this new function to processors/processor.go
func GetSchema(filePath string) ([]string, error) {
	fileExtension := strings.ToLower(filepath.Ext(filePath))
	var records [][]string

	switch fileExtension {
	case ".csv":
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		reader := csv.NewReader(file)
		records, err = reader.ReadAll()
		if err != nil {
			return nil, err
		}
	case ".xlsx":
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		sheetName := f.GetSheetName(0) // Get the first sheet
		records, err = f.GetRows(sheetName)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported file type for schema detection: %s", fileExtension)
	}

	if len(records) > 0 {
		return records[0], nil // Return the header row
	}

	return []string{}, nil
}