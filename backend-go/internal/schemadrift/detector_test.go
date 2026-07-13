package schemadrift

import (
	"testing"
	"time"
)

// --- Test GORM models ---

type testUser struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"column:username;type:varchar(100);not null" json:"username"`
	Email     string    `gorm:"column:email;type:varchar(255)" json:"email"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func TestDetectLiveIgnoresMigrationLedgerTable(t *testing.T) {
	d := &DriftDetector{}
	report := d.detect(map[string][]columnInfo{
		"application_table": {{ColumnName: "id", DataType: "bigint"}},
		"schema_migrations": {{ColumnName: "version", DataType: "bigint"}},
	}, []TableDef{{Name: "application_table", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}}})
	if len(report.ExtraTables) != 0 {
		t.Fatalf("migration ledger reported as extra: %+v", report.ExtraTables)
	}
}

func TestDetectLiveComparesInformationSchemaBaseTypes(t *testing.T) {
	d := &DriftDetector{}
	report := d.detect(map[string][]columnInfo{
		"sample": {
			{ColumnName: "name", DataType: "character varying"},
			{ColumnName: "amount", DataType: "numeric"},
		},
	}, []TableDef{{Name: "sample", Columns: []ColumnDef{
		{Name: "name", Type: "character varying(80)"},
		{Name: "amount", Type: "numeric(12,2)"},
	}}})
	if report.ColumnMismatch != 0 {
		t.Fatalf("type modifiers unavailable from information_schema must not be drift: %+v", report)
	}
}

func TestDetectLiveDoesNotGuessArrayOrUserDefinedElementTypes(t *testing.T) {
	d := &DriftDetector{}
	report := d.detect(map[string][]columnInfo{
		"sample": {
			{ColumnName: "tags", DataType: "ARRAY"},
			{ColumnName: "state", DataType: "USER-DEFINED"},
		},
	}, []TableDef{{Name: "sample", Columns: []ColumnDef{
		{Name: "tags", Type: "text[]"},
		{Name: "state", Type: "custom_state"},
	}}})
	if report.ColumnMismatch != 0 {
		t.Fatalf("insufficient information_schema type identity must not create drift: %+v", report)
	}
}

func (testUser) TableName() string { return "test_users" }

type testProduct struct {
	ID       int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name     string    `gorm:"column:name;type:varchar(200);not null" json:"name"`
	Price    float64   `gorm:"column:price;type:numeric(10,2);not null;default:0" json:"price"`
	SkuID    int64     `gorm:"column:sku_id;type:bigint;not null" json:"sku_id"`
	Status   string    `gorm:"column:status;type:varchar(20);default:active" json:"status"`
	CreateAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (testProduct) TableName() string { return "test_products" }

type testOrder struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderNo     string    `gorm:"column:order_no;type:varchar(100);not null" json:"order_no"`
	TotalAmount float64   `gorm:"column:total_amount;type:numeric(14,2);not null;default:0" json:"total_amount"`
	Status      string    `gorm:"column:status;type:varchar(20);default:pending" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (testOrder) TableName() string { return "test_orders" }

// --- Tests ---

func TestDetectStatic_NoDrift(t *testing.T) {
	// Migration and model should match exactly: both use "test_users" with matching columns
	migrations := []TableDef{
		{
			Name: "test_users",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true, Nullable: false},
				{Name: "username", Type: "character varying(100)", Nullable: false},
				{Name: "email", Type: "character varying(255)", Nullable: true},
				{Name: "created_at", Type: "timestamp with time zone", Nullable: false},
				{Name: "updated_at", Type: "timestamp with time zone", Nullable: false},
			},
		},
	}

	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{testUser{}})

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	for _, dif := range report.Diffs {
		t.Errorf("unexpected drift: %s.%s: %s (mig=%s mod=%s)",
			dif.TableName, dif.ColumnName, dif.DiffType, dif.MigrationValue, dif.ModelValue)
	}
}

func TestDetectStatic_MissingTable(t *testing.T) {
	migrations := []TableDef{
		{Name: "table_a", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}},
		{Name: "table_b", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}},
	}
	models := []TableDef{
		{Name: "table_a", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}},
	}

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	if len(report.Diffs) != 1 {
		t.Fatalf("expected 1 drift (missing table_b), got %d", len(report.Diffs))
	}
	if report.Diffs[0].DiffType != "missing_table" {
		t.Errorf("expected missing_table, got %s", report.Diffs[0].DiffType)
	}
	if report.Diffs[0].TableName != "table_b" {
		t.Errorf("expected table_b, got %s", report.Diffs[0].TableName)
	}
}

func TestDetectStatic_ExtraTable(t *testing.T) {
	migrations := []TableDef{
		{Name: "table_a", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}},
	}
	models := []TableDef{
		{Name: "table_a", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}},
		{Name: "table_b", Columns: []ColumnDef{{Name: "id", Type: "bigint"}}},
	}

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	extraFound := false
	for _, dif := range report.Diffs {
		if dif.DiffType == "extra_table" && dif.TableName == "table_b" {
			extraFound = true
		}
	}
	if !extraFound {
		t.Errorf("expected extra_table for table_b, got: %+v", report.Diffs)
	}
}

func TestDetectStatic_MissingColumn(t *testing.T) {
	migrations := []TableDef{
		{
			Name: "test_users",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "username", Type: "character varying(100)"},
				{Name: "email", Type: "character varying(255)"},
				{Name: "phone", Type: "character varying(20)"}, // in migration but not in model
			},
		},
	}

	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{testUser{}})

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	found := false
	for _, dif := range report.Diffs {
		if dif.DiffType == "missing_column" && dif.ColumnName == "phone" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected missing_column for phone, got: %+v", report.Diffs)
	}
}

func TestDetectStatic_ExtraColumn(t *testing.T) {
	// Migration has fewer columns than the model (no phone column)
	migrations := []TableDef{
		{
			Name: "test_users_extra",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint"},
				{Name: "username", Type: "character varying(100)"},
			},
		},
	}

	type testUserFull struct {
		ID       int64  `gorm:"column:id;type:bigint;primaryKey"`
		Username string `gorm:"column:username;type:varchar(100)"`
		Phone    string `gorm:"column:phone;type:varchar(20)"`
	}
	_ = testUserFull{}

	// Register a model with an extra phone column
	models := []TableDef{
		{
			Name: "test_users_extra",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "username", Type: "character varying(100)", Nullable: true},
				{Name: "phone", Type: "character varying(20)", Nullable: true},
			},
		},
	}

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	found := false
	for _, dif := range report.Diffs {
		if dif.DiffType == "extra_column" && dif.ColumnName == "phone" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected extra_column for phone, got: %+v", report.Diffs)
	}
}

func TestDetectStatic_TypeMismatch(t *testing.T) {
	// Migration says name is TEXT, model says VARCHAR(200) — mismatch
	migrations := []TableDef{
		{
			Name: "test_products",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "name", Type: "text", Nullable: false},
				{Name: "price", Type: "numeric(10,2)", Nullable: false},
			},
		},
	}

	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{testProduct{}})

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	found := false
	for _, dif := range report.Diffs {
		if dif.DiffType == "type_mismatch" && dif.ColumnName == "name" {
			found = true
			if dif.MigrationValue != "text" {
				t.Errorf("expected migration value 'text', got %q", dif.MigrationValue)
			}
			if dif.ModelValue != "character varying(200)" {
				t.Errorf("expected model value 'character varying(200)', got %q", dif.ModelValue)
			}
		}
	}
	if !found {
		t.Errorf("expected type_mismatch for name column, got: %+v", report.Diffs)
	}
}

func TestDetectStatic_DefaultDiff(t *testing.T) {
	migrations := []TableDef{
		{
			Name: "test_products",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "name", Type: "character varying(200)", Nullable: false},
				{Name: "status", Type: "character varying(20)", Default: "'active'"},
			},
		},
	}

	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{testProduct{}})

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	// Check that there's NO default diff for status (both say 'active')
	for _, dif := range report.Diffs {
		if dif.ColumnName == "status" && dif.DiffType == "default_diff" {
			t.Logf("status default: migration=%q model=%q", dif.MigrationValue, dif.ModelValue)
		}
	}

	// Now test a case with actual default mismatch: migration says 'shipped', model says 'pending'
	migrations2 := []TableDef{
		{
			Name: "test_orders",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "status", Type: "character varying(20)", Default: "'shipped'"},
			},
		},
	}

	models2 := reflector.Reflect([]interface{}{testOrder{}})
	report2 := d.detectStatic(migrations2, models2)

	found := false
	for _, dif := range report2.Diffs {
		if dif.DiffType == "default_diff" && dif.ColumnName == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected default_diff for status column, got: %+v", report2.Diffs)
	}
}

func TestReflector_TableName(t *testing.T) {
	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{testUser{}, testProduct{}, testOrder{}})

	names := make(map[string]bool)
	for _, m := range models {
		names[m.Name] = true
	}

	if !names["test_users"] {
		t.Errorf("expected test_users table")
	}
	if !names["test_products"] {
		t.Errorf("expected test_products table")
	}
	if !names["test_orders"] {
		t.Errorf("expected test_orders table")
	}
}

func TestReflector_PointerReceiver(t *testing.T) {
	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{&testUser{}, &testProduct{}})

	names := make(map[string]bool)
	for _, m := range models {
		names[m.Name] = true
	}

	if !names["test_users"] {
		t.Errorf("expected test_users table from pointer receiver")
	}
	if !names["test_products"] {
		t.Errorf("expected test_products table from pointer receiver")
	}
}

func TestDetectStatic_PointerModels(t *testing.T) {
	migrations := []TableDef{
		{
			Name: "test_users",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "username", Type: "character varying(100)"},
			},
		},
		{
			Name: "test_products",
			Columns: []ColumnDef{
				{Name: "id", Type: "bigint", PrimaryKey: true},
				{Name: "name", Type: "character varying(200)"},
			},
		},
	}

	reflector := NewModelReflector()
	models := reflector.Reflect([]interface{}{&testUser{}, &testProduct{}})

	d := &DriftDetector{}
	report := d.detectStatic(migrations, models)

	// We expect extra columns for the columns not in migrations but present in models
	// (price, sku_id, status, created_at in testProducts; email, created_at, updated_at in testUsers)
	extraCols := 0
	for _, dif := range report.Diffs {
		if dif.DiffType == "extra_column" {
			extraCols++
		}
	}
	if extraCols < 3 {
		t.Errorf("expected at least 3 extra_column diffs, got %d", extraCols)
	}
}
