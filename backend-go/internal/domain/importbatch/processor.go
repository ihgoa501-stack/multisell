package importbatch

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProcessBatchDispatcher decodes rows from a parsed file into ImportBatchRows
// and dispatches them to type-specific handlers.
type ProcessBatchDispatcher struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewProcessBatchDispatcher creates a new processor.
func NewProcessBatchDispatcher(db *gorm.DB, logger *zap.Logger) *ProcessBatchDispatcher {
	return &ProcessBatchDispatcher{db: db, logger: logger}
}

// ProcessBatch reads a batch's uploaded file, parses it, and dispatches
// each row to the appropriate handler based on batch type.
func (p *ProcessBatchDispatcher) ProcessBatch(batchID int64) error {
	// Fetch the batch record.
	var batch ImportBatch
	if err := p.db.First(&batch, batchID).Error; err != nil {
		return fmt.Errorf("fetch batch %d: %w", batchID, err)
	}

	// Locate the file.
	filePath := filepath.Join("uploads", batch.FileName)
	format, err := DetectFormat(batch.FileName)
	if err != nil {
		return p.failBatch(&batch, err.Error())
	}

	// Parse the file.
	var rows []map[string]string
	switch format {
	case "xlsx":
		rows, err = ParseExcel(filePath)
	case "csv":
		rows, err = ParseCSV(filePath)
	}
	if err != nil {
		return p.failBatch(&batch, fmt.Sprintf("parse file: %v", err))
	}

	// Update batch with total row count.
	batch.TotalRows = len(rows)
	batch.Status = "processing"
	p.db.Save(&batch)

	// Process each row.
	successCount := 0
	errorCount := 0
	var firstErrors []string

	for i, row := range rows {
		rawJSON, _ := json.Marshal(row)
		rowRec := &ImportBatchRow{
			BatchID:  batch.ID,
			RowIndex: i + 1,
			Status:   "pending",
			RawData:  rawJSON,
		}

		err := p.dispatchRow(batch.SourceType, row)
		if err != nil {
			rowRec.Status = "failed"
			rowRec.ErrorMessage = err.Error()
			errorCount++
			if len(firstErrors) < 5 {
				firstErrors = append(firstErrors, fmt.Sprintf("row %d: %v", i+1, err))
			}
		} else {
			rowRec.Status = "success"
			successCount++
		}

		if err := p.db.Create(rowRec).Error; err != nil {
			p.logger.Error("create import batch row", zap.Error(err))
		}
	}

	// Finalize batch.
	batch.SuccessCount = successCount
	batch.ErrorCount = errorCount
	if errorCount > 0 {
		batch.Status = "partial"
		if len(firstErrors) > 0 {
			summary, _ := json.Marshal(firstErrors)
			batch.ErrorSummary = string(summary)
		}
	} else {
		batch.Status = "completed"
	}
	return p.db.Save(&batch).Error
}

// dispatchRow routes a parsed row to the type-specific handler.
func (p *ProcessBatchDispatcher) dispatchRow(batchType string, row map[string]string) error {
	switch batchType {
	case "product":
		return p.handleProductRow(row)
	case "order":
		return p.handleOrderRow(row)
	case "inventory":
		return p.handleInventoryRow(row)
	default:
		return fmt.Errorf("unsupported batch type: %s", batchType)
	}
}

// handleProductRow validates and processes a product import row.
func (p *ProcessBatchDispatcher) handleProductRow(row map[string]string) error {
	if row["name"] == "" {
		return fmt.Errorf("missing required field: name")
	}
	if row["sku"] == "" {
		return fmt.Errorf("missing required field: sku")
	}
	// TODO: actual product creation/update logic
	p.logger.Debug("product row processed", zap.String("sku", row["sku"]), zap.String("name", row["name"]))
	return nil
}

// handleOrderRow validates and processes an order import row.
func (p *ProcessBatchDispatcher) handleOrderRow(row map[string]string) error {
	if row["order_no"] == "" {
		return fmt.Errorf("missing required field: order_no")
	}
	// TODO: actual order creation/update logic
	p.logger.Debug("order row processed", zap.String("order_no", row["order_no"]))
	return nil
}

// handleInventoryRow validates and processes an inventory import row.
func (p *ProcessBatchDispatcher) handleInventoryRow(row map[string]string) error {
	if row["sku"] == "" {
		return fmt.Errorf("missing required field: sku")
	}
	// TODO: actual inventory update logic
	p.logger.Debug("inventory row processed", zap.String("sku", row["sku"]))
	return nil
}

// failBatch marks a batch as failed with the given error message.
func (p *ProcessBatchDispatcher) failBatch(batch *ImportBatch, msg string) error {
	batch.Status = "failed"
	batch.ErrorSummary = msg
	return p.db.Save(batch).Error
}
