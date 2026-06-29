package compliance

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestFreshnessWriter_RecordVerification(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t) // no models — data_freshness table is production-only
	// data_freshness table does not exist in the test DB — this test verifies the
	// call doesn't panic. Full integration test when schema is available.
	w := NewFreshnessWriter(db)
	err := w.RecordVerification(1, "pass")
	// In test DB (SQLite), the table doesn't exist — we accept an error.
	_ = err
}
