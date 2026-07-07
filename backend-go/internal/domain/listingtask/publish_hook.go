package listingtask

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PublishHook is an optional callback called after ExecuteTask successfully
// transitions a listing task to "completed". The hook calls the platform
// adapter's Publish to push the product live. If it returns an error, the
// task status reverts to "failed" so the operator can retry.
//
// mode is the ExecutionMode (0=dry_run, 1=sandbox, 2=approval_required, 3=production).
// The hook should pass this mode through context to the platform adapter.
//
// ponytail: idempotency key ("listing_task:<id>:<mode>:<created_at>") prevents duplicate publishes.
// Upgrade to EventBus + outbox when cross-service idempotency matters.
type PublishHook func(taskID int64, mode ExecutionMode) error

// NewPublishHook creates the default PublishHook that:
//  1. Loads the listing task
//  2. Resolves the platform adapter
//  3. Loads product data and active integration account
//  4. Sets execution mode in context
//  5. Calls adapter.Publish with the constructed PublishInput
//  6. Saves external_reference_id and external_reference_url to the task
//  7. Writes an audit log entry
//
// ponytail: single-adapter resolve, no multi-store routing. Add account_id to
// ListingTask if multi-store routing becomes needed.
func NewPublishHook(db *gorm.DB, auditSvc *operationlog.Service, logger *zap.Logger) PublishHook {
	return func(taskID int64, mode ExecutionMode) error {
		// 1. Load the listing task.
		var task ListingTask
		if err := db.First(&task, taskID).Error; err != nil {
			return fmt.Errorf("publish hook: load task %d: %w", taskID, err)
		}

		// 2. Resolve platform code.
		type platRow struct {
			Code string
		}
		var plat platRow
		if err := db.Table("platform").
			Select("code").
			Where("id = ?", task.PlatformID).
			Scan(&plat).Error; err != nil {
			return fmt.Errorf("publish hook: load platform %d: %w", task.PlatformID, err)
		}

		// 3. Get the platform adapter.
		adapter, ok := integrations.GetAdapter(plat.Code)
		if !ok {
			return fmt.Errorf("publish hook: no adapter for platform %q", plat.Code)
		}

		// 4. Load product and SKU data.
		var prod sku.Product
		if err := db.First(&prod, task.ProductID).Error; err != nil {
			return fmt.Errorf("publish hook: load product %d: %w", task.ProductID, err)
		}
		type skuRow struct {
			ID    int64
			Code  string
			Stock int
		}
		var skus []skuRow
		if err := db.Table("sku").
			Select("id, code, stock").
			Where("product_id = ?", task.ProductID).
			Find(&skus).Error; err != nil {
			return fmt.Errorf("publish hook: load SKUs for product %d: %w", task.ProductID, err)
		}

		// 5. Find first active account for this platform.
		var acct integrations.PlatformIntegrationAccount
		if err := db.Where("platform_id = ? AND status = ?", task.PlatformID, "active").
			First(&acct).Error; err != nil {
			return fmt.Errorf("publish hook: no active account for platform %d: %w", task.PlatformID, err)
		}

		// 6. Build PublishInput with idempotency key to prevent duplicate publishes.
		// Idempotency key = "listing_task:<id>:<mode>:<created_at_unix>"
		ikey := "listing_task:" + strconv.FormatInt(task.ID, 10) +
			":" + strconv.FormatInt(int64(mode), 10) +
			":" + strconv.FormatInt(task.CreatedAt.Unix(), 10)
		prices := make(map[int64]string)
		inventories := make(map[int64]int)
		publishSKUs := make([]integrations.PublishSKU, 0, len(skus))
		for _, sk := range skus {
			publishSKUs = append(publishSKUs, integrations.PublishSKU{SkuID: sk.ID, SkuCode: sk.Code})
			if task.TargetSalePrice != nil {
				prices[sk.ID] = fmt.Sprintf("%.2f", *task.TargetSalePrice)
			}
			inventories[sk.ID] = sk.Stock
		}
		pkgH, _ := prod.PackageHeightCm.Float64()
		pkgW, _ := prod.PackageWidthCm.Float64()
		pkgL, _ := prod.PackageLengthCm.Float64()
		pkgWt, _ := prod.PackageWeightKg.Float64()

		// 7. Set execution mode in context before calling the adapter.
		ctx := integrations.WithExecutionMode(context.Background(), integrations.ExecutionMode(mode))

		result, err := adapter.Publish(ctx, &integrations.PublishInput{
			ProductID:     task.ProductID,
			PlatformID:    task.PlatformID,
			AccountID:     acct.ID,
			SKUs:          publishSKUs,
			Prices:        prices,
			Inventories:   inventories,
			IdempotencyKey: ikey,
			ProductName:   prod.Name,
			Description:   prod.Description,
			CategoryID:    prod.CategoryID,
			MainImage:     prod.MainImage,
			PackageHeight: pkgH,
			PackageWidth:  pkgW,
			PackageLength: pkgL,
			PackageWeight: pkgWt,
		})

		// 8. Write audit log entry.
		auditAction := "publish"
		if mode == ExecutionModeSandbox {
			auditAction = "sandbox_publish"
		}
		auditContent := fmt.Sprintf("publish product %d to platform %s (account %d) mode=%s",
			task.ProductID, plat.Code, acct.ID, ExecutionModeNames[mode])
		if err != nil {
			auditContent = fmt.Sprintf("publish product %d to platform %s failed: %v mode=%s",
				task.ProductID, plat.Code, err, ExecutionModeNames[mode])
		}
		if auditSvc != nil {
			auditSvc.LogStructured(&operationlog.StructuredLogInput{
				Module:      "platform_publish",
				Action:      auditAction,
				ResourceID:  fmt.Sprintf("listing_task:%d", taskID),
				Operator:    "system",
				Content:     auditContent,
				Result: func() string {
					if err != nil {
						return "failure"
					}
					return "success"
				}(),
				TriggerType: "system",
				EntityType:  "listing_task",
				EntityID:    taskID,
			})
		}

		if err != nil {
			return fmt.Errorf("publish hook: platform %s publish failed: %w", plat.Code, err)
		}

		// 9. Merge platform publish result into existing item result.
		resultJSON, _ := json.Marshal(result)
		db.Exec(`UPDATE listing_task_item SET result = COALESCE(result, '{}')::jsonb || ?::jsonb WHERE task_id = ?`,
			string(resultJSON), taskID)

		// 10. Save external reference ID and URL to the listing task.
		updates := map[string]interface{}{}
		if result.PlatformProductID != "" {
			updates["external_reference_id"] = result.PlatformProductID
		}
		if result.PlatformURL != "" {
			updates["external_reference_url"] = result.PlatformURL
		}
		if len(updates) > 0 {
			if err := db.Model(&task).Updates(updates).Error; err != nil {
				logger.Warn("publish hook: failed to save external reference",
					zap.Int64("task_id", taskID),
					zap.Error(err),
				)
			}
		}

		logger.Info("listing task platform publish succeeded",
			zap.Int64("task_id", taskID),
			zap.String("platform", plat.Code),
			zap.String("mode", ExecutionModeNames[mode]),
			zap.String("platform_product_id", result.PlatformProductID),
			zap.String("platform_url", result.PlatformURL),
		)
		return nil
	}
}
