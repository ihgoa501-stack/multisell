package sourcing1688

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestPostgresCaptureRespectsImmutableSnapshot runs only when a disposable,
// fully migrated PostgreSQL DSN is provided. It guards behavior SQLite cannot
// model faithfully: migration 000084 rejects every snapshot UPDATE/DELETE.
func TestPostgresCaptureRespectsImmutableSnapshot(t *testing.T) {
	dsn := os.Getenv("SOURCING1688_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SOURCING1688_POSTGRES_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	statements := []string{
		`INSERT INTO "user" (id, username, password_hash, role) VALUES (42001, 'sourcing-pg-owner', 'not-used', 'admin')`,
		`INSERT INTO demand_case (id, owner_id, region, consumer, need_scenario, sales_channel, status) VALUES (42001, 42001, 'test-region', 'test-consumer', 'test-scenario', 'test-channel', 'experiment_ready')`,
		`INSERT INTO experiment_case (experiment_id, name, stage, status, owner_id) VALUES ('EXP-PG-IMMUTABLE', 'postgres immutable capture', 'product', 'active', 42001)`,
		`INSERT INTO experiment_gate_decision (experiment_id, stage, gate_code, result, decided_by) VALUES ('EXP-PG-IMMUTABLE', 'opportunity', 'opportunity', 'pass', 42001)`,
		`INSERT INTO experiment_object_link (experiment_id, object_type, object_id) VALUES ('EXP-PG-IMMUTABLE', 'demand_case', '42001')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed postgres fixture: %v", err)
		}
	}
	title := "PostgreSQL immutable snapshot product"
	rawPayload := json.RawMessage("{\n  \"z\": 1, \"a\": 2\n}")
	product, err := NewService(db, zap.NewNop()).Capture(&CaptureInput{
		DemandCaseID: 42001, ExperimentID: "EXP-PG-IMMUTABLE", SourceURL: "https://detail.1688.com/offer/42001001.html",
		CollectedAt: time.Now().UTC(), CollectedBy: 42001, Driver: "plugin", ParserVersion: "integration@1",
		SupplierBusinessID: "supplier-pg-42001", RawPayload: rawPayload, Title: &title,
		SkuVariants: json.RawMessage(`[{"color":"red"}]`), CaptureMode: CaptureModeControlledFetch, CollectionRequestID: "req_postgres-42001",
	})
	if err != nil {
		t.Fatalf("Capture failed under PostgreSQL immutable trigger: %v", err)
	}
	var snapshot Sourcing1688Snapshot
	if product.SnapshotID == nil || db.First(&snapshot, *product.SnapshotID).Error != nil || snapshot.ProductFingerprint == "" || snapshot.ProductFingerprint != product.SourceProductFingerprint {
		t.Fatalf("fingerprint not atomically inserted: product=%#v snapshot=%#v", product, snapshot)
	}
	expectedHash := sha256.Sum256(rawPayload)
	if !bytes.Equal(snapshot.RawPayload, rawPayload) || snapshot.RawSHA256 != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("original collector bytes or SHA changed: got=%q sha=%s", snapshot.RawPayload, snapshot.RawSHA256)
	}
}
