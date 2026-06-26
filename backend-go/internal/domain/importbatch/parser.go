package importbatch

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// DetectFormat returns the file format based on the extension.
// Returns "xlsx" for .xlsx, "csv" for .csv, or an error for unsupported formats.
func DetectFormat(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx":
		return "xlsx", nil
	case ".csv":
		return "csv", nil
	default:
		return "", fmt.Errorf("unsupported file format: %s (supported: .xlsx, .csv)", ext)
	}
}

// ParseExcel reads an .xlsx file and returns each row as a map keyed by column header.
func ParseExcel(filePath string) ([]map[string]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer f.Close()

	// Use the first sheet.
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel file has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("excel file has no data rows")
	}

	headers := rows[0]
	var result []map[string]string
	for i := 1; i < len(rows); i++ {
		row := make(map[string]string, len(headers))
		for j, h := range headers {
			val := ""
			if j < len(rows[i]) {
				val = rows[i][j]
			}
			row[h] = val
		}
		result = append(result, row)
	}
	return result, nil
}

// ParseCSV reads a CSV file and returns each row as a map keyed by column header.
func ParseCSV(filePath string) ([]map[string]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open csv file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("csv file has no data rows")
	}

	headers := records[0]
	var result []map[string]string
	for i := 1; i < len(records); i++ {
		row := make(map[string]string, len(headers))
		for j, h := range headers {
			val := ""
			if j < len(records[i]) {
				val = records[i][j]
			}
			row[h] = val
		}
		result = append(result, row)
	}
	return result, nil
}
