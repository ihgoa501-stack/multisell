package demandcase

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestPostgresResearchProvenanceConstraints runs only against a disposable,
// fully migrated PostgreSQL database. Migration 000106 uses PostgreSQL
// triggers and foreign keys whose behavior SQLite cannot represent.
func TestPostgresResearchProvenanceConstraints(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; provide a disposable database migrated through 000106")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatalf("begin fixture transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	nonce := time.Now().UnixNano()
	var batchOwner1, batchOwner2, caseOwner1, caseOwner2, snapshotOwner1 int64
	mustQueryID(t, tx, `INSERT INTO demand_research_batch (batch_key, owner_id) VALUES ($1, $2) RETURNING id`, fmt.Sprintf("pg-constraints-owner1-%d", nonce), int64(810001), &batchOwner1)
	mustQueryID(t, tx, `INSERT INTO demand_research_batch (batch_key, owner_id) VALUES ($1, $2) RETURNING id`, fmt.Sprintf("pg-constraints-owner2-%d", nonce), int64(810002), &batchOwner2)
	mustQueryID(t, tx, `INSERT INTO demand_case (owner_id, region, consumer, need_scenario, sales_channel, target_locale) VALUES ($1, 'r1', 'c1', 'n1', 'ch1', 'en-US') RETURNING id`, int64(810001), &caseOwner1)
	mustQueryID(t, tx, `INSERT INTO demand_case (owner_id, region, consumer, need_scenario, sales_channel, target_locale) VALUES ($1, 'r2', 'c2', 'n2', 'ch2', 'en-US') RETURNING id`, int64(810002), &caseOwner2)
	collectedAt := time.Now().UTC().Truncate(time.Microsecond)
	mustQueryID(t, tx, `INSERT INTO demand_research_snapshot (batch_id, owner_id, demand_case_id, run_id, run_type, source_uri, collected_at, raw_payload, raw_sha256) VALUES ($1, $2, $3, 'run-owner1', 'scout_result', 'https://example.test/source-1', $4, '{}', repeat('a', 64)) RETURNING id`, batchOwner1, int64(810001), caseOwner1, collectedAt, &snapshotOwner1)

	if _, err := tx.Exec(`INSERT INTO demand_evidence (demand_case_id, dimension, kind, truth_status, title, source_uri, observed_at, run_id, snapshot_id) VALUES ($1, 'demand', 'support', 'quoted', 'valid binding', 'https://example.test/source-1', $2, 'run-owner1', $3)`, caseOwner1, collectedAt, snapshotOwner1); err != nil {
		t.Fatalf("valid provenance binding rejected: %v", err)
	}

	expectPostgresReject(t, tx, "cross-Owner snapshot", `INSERT INTO demand_research_snapshot (batch_id, owner_id, demand_case_id, run_id, run_type, source_uri, collected_at, raw_payload, raw_sha256) VALUES ($1, $2, $3, 'run-cross-owner', 'scout_result', 'https://example.test/cross-owner', $4, '{}', repeat('b', 64))`, batchOwner1, int64(810001), caseOwner2, collectedAt)
	expectPostgresReject(t, tx, "cross-case evidence", `INSERT INTO demand_evidence (demand_case_id, dimension, kind, truth_status, title, source_uri, observed_at, run_id, snapshot_id) VALUES ($1, 'demand', 'support', 'quoted', 'cross case', 'https://example.test/source-1', $2, 'run-owner1', $3)`, caseOwner2, collectedAt, snapshotOwner1)
	expectPostgresReject(t, tx, "mismatched run", `INSERT INTO demand_evidence (demand_case_id, dimension, kind, truth_status, title, source_uri, observed_at, run_id, snapshot_id) VALUES ($1, 'demand', 'support', 'quoted', 'wrong run', 'https://example.test/source-1', $2, 'different-run', $3)`, caseOwner1, collectedAt, snapshotOwner1)
	expectPostgresReject(t, tx, "case Owner tampering", `UPDATE demand_case SET owner_id = $1 WHERE id = $2`, int64(810002), caseOwner1)
	expectPostgresReject(t, tx, "batch Owner tampering", `UPDATE demand_research_batch SET owner_id = $1 WHERE id = $2`, int64(810002), batchOwner1)
	expectPostgresReject(t, tx, "snapshot tampering", `UPDATE demand_research_snapshot SET raw_payload = '{"tampered":true}' WHERE id = $1`, snapshotOwner1)
	expectPostgresReject(t, tx, "snapshot deletion", `DELETE FROM demand_research_snapshot WHERE id = $1`, snapshotOwner1)

	_ = batchOwner2 // owner-2 batch is deliberately present to model distinct scope.
}

func mustQueryID(t *testing.T, tx *sql.Tx, query string, args ...any) {
	t.Helper()
	destination := args[len(args)-1].(*int64)
	if err := tx.QueryRow(query, args[:len(args)-1]...).Scan(destination); err != nil {
		t.Fatalf("seed postgres fixture: %v", err)
	}
}

func expectPostgresReject(t *testing.T, tx *sql.Tx, name, query string, args ...any) {
	t.Helper()
	savepoint := "constraint_probe"
	if _, err := tx.Exec("SAVEPOINT " + savepoint); err != nil {
		t.Fatalf("%s: create savepoint: %v", name, err)
	}
	if _, err := tx.Exec(query, args...); err == nil {
		t.Fatalf("%s: unsafe write unexpectedly succeeded", name)
	}
	if _, err := tx.Exec("ROLLBACK TO SAVEPOINT " + savepoint); err != nil {
		t.Fatalf("%s: recover savepoint: %v", name, err)
	}
	if _, err := tx.Exec("RELEASE SAVEPOINT " + savepoint); err != nil {
		t.Fatalf("%s: release savepoint: %v", name, err)
	}
}
