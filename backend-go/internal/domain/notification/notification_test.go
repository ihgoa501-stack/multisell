package notification

import (
	"fmt"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_Notification_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	n := &Notification{
		UserID:    1,
		AlertType: "low_stock",
		Title:     "库存不足",
		Content:   "SKU #1001 库存仅剩5件",
		Severity:  "warning",
	}
	err := svc.Create(n)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.ID == 0 {
		t.Fatal("ID should be set")
	}

	got, err := svc.GetByID(n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "库存不足" {
		t.Fatalf("Title = %s", got.Title)
	}
}

func TestService_Notification_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Notification{UserID: 1, AlertType: "low_stock", Title: "T1", Severity: "warning"})
	db.Create(&Notification{UserID: 1, AlertType: "price_change", Title: "T2", Severity: "info"})
	db.Create(&Notification{UserID: 2, AlertType: "low_stock", Title: "T3", Severity: "warning"})

	// List for user 1
	items, total, err := svc.List(ListFilter{UserID: 1}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d (expected 2)", total)
	}
	_ = items

	// Filter by alert type and severity
	items, total, err = svc.List(ListFilter{UserID: 1, AlertType: "low_stock", Severity: "warning"}, 1, 10)
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d (expected 1)", total)
	}
}

func TestService_Notification_MarkReadAndDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&Notification{UserID: 1, AlertType: "test", Title: "T1", IsRead: 0})
	db.Create(&Notification{UserID: 1, AlertType: "test", Title: "T2", IsRead: 0})

	// Mark single
	err := svc.MarkAsRead(1)
	if err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}
	n, _ := svc.GetByID(1)
	if n.IsRead != 1 {
		t.Fatalf("IsRead = %d (expected 1)", n.IsRead)
	}

	// Mark all read
	err = svc.MarkAllRead(1)
	if err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	n2, _ := svc.GetByID(2)
	if n2.IsRead != 1 {
		t.Fatalf("IsRead = %d (expected 1)", n2.IsRead)
	}

	// UnreadCount
	cnt, err := svc.UnreadCount(1)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("unread = %d (expected 0)", cnt)
	}

	// Delete
	err = svc.Delete(1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(1)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_AlertRule_CreateAndList(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	r := &AlertRule{
		Name:      "低库存预警",
		AlertType: "low_stock",
		Enabled:   1,
	}
	err := svc.CreateAlertRule(r)
	if err != nil {
		t.Fatalf("CreateAlertRule: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("ID should be set")
	}

	rules, err := svc.ListAlertRules()
	if err != nil {
		t.Fatalf("ListAlertRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len = %d (expected 1)", len(rules))
	}

	got, err := svc.GetAlertRule(r.ID)
	if err != nil {
		t.Fatalf("GetAlertRule: %v", err)
	}
	if got.Name != "低库存预警" {
		t.Fatalf("Name = %s", got.Name)
	}
}

func TestList_Pagination(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create 25 notifications for user 1
	for i := 0; i < 25; i++ {
		db.Create(&Notification{UserID: 1, AlertType: "test", Title: fmt.Sprintf("T%d", i+1), Severity: "info"})
	}

	// Page 1, size 10
	items, total, err := svc.List(ListFilter{UserID: 1}, 1, 10)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 25 {
		t.Fatalf("total = %d (expected 25)", total)
	}
	if len(items) != 10 {
		t.Fatalf("page 1 len = %d (expected 10)", len(items))
	}

	// Page 3, size 10 → should return the last 5 items
	items, total, err = svc.List(ListFilter{UserID: 1}, 3, 10)
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("page 3 len = %d (expected 5)", len(items))
	}

	// Default size when size < 1
	items, total, err = svc.List(ListFilter{UserID: 1}, 1, 0)
	if err != nil {
		t.Fatalf("List default size: %v", err)
	}
	if len(items) != 20 {
		t.Fatalf("default size len = %d (expected 20)", len(items))
	}
}

func TestDelete_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Deleting a non-existent ID should not error (GORM behavior)
	err := svc.Delete(99999)
	if err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}

	// Verify existing records are unaffected
	svc.Create(&Notification{UserID: 1, AlertType: "test", Title: "T1"})
	items, total, _ := svc.List(ListFilter{}, 1, 10)
	if total != 1 {
		t.Fatalf("total = %d (expected 1)", total)
	}
	_ = items
}

func TestAlertRule_ToggleEnabled(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	r := &AlertRule{Name: "低库存预警", AlertType: "low_stock", Enabled: 0}
	if err := svc.CreateAlertRule(r); err != nil {
		t.Fatalf("CreateAlertRule: %v", err)
	}

	// Toggle disabled → enabled
	r.Enabled = 1
	if err := svc.UpdateAlertRule(r); err != nil {
		t.Fatalf("UpdateAlertRule (enable): %v", err)
	}
	got, _ := svc.GetAlertRule(r.ID)
	if got.Enabled != 1 {
		t.Fatalf("Enabled = %d (expected 1)", got.Enabled)
	}

	// Toggle enabled → disabled
	r.Enabled = 0
	if err := svc.UpdateAlertRule(r); err != nil {
		t.Fatalf("UpdateAlertRule (disable): %v", err)
	}
	got, _ = svc.GetAlertRule(r.ID)
	if got.Enabled != 0 {
		t.Fatalf("Enabled = %d (expected 0)", got.Enabled)
	}
}

func TestUnreadCount_MultipleUsers(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	// User 1: 3 unread + 1 read
	// User 2: 1 unread
	db.Create(&Notification{UserID: 1, AlertType: "test", Title: "U1a", IsRead: 0})
	db.Create(&Notification{UserID: 1, AlertType: "test", Title: "U1b", IsRead: 0})
	db.Create(&Notification{UserID: 1, AlertType: "test", Title: "U1c", IsRead: 0})
	db.Create(&Notification{UserID: 1, AlertType: "test", Title: "U1d", IsRead: 1})
	db.Create(&Notification{UserID: 2, AlertType: "test", Title: "U2a", IsRead: 0})

	cnt, err := svc.UnreadCount(1)
	if err != nil {
		t.Fatalf("UnreadCount user 1: %v", err)
	}
	if cnt != 3 {
		t.Fatalf("user 1 unread = %d (expected 3)", cnt)
	}

	cnt, err = svc.UnreadCount(2)
	if err != nil {
		t.Fatalf("UnreadCount user 2: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("user 2 unread = %d (expected 1)", cnt)
	}
}

func TestService_AlertRule_UpdateAndDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Notification{}, &AlertRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreateAlertRule(&AlertRule{Name: "Rule1", AlertType: "type1"})
	rules, _ := svc.ListAlertRules()
	r := rules[0]
	r.Name = "Rule1_updated"
	err := svc.UpdateAlertRule(&r)
	if err != nil {
		t.Fatalf("UpdateAlertRule: %v", err)
	}

	got, _ := svc.GetAlertRule(r.ID)
	if got.Name != "Rule1_updated" {
		t.Fatalf("Name = %s", got.Name)
	}

	err = svc.DeleteAlertRule(r.ID)
	if err != nil {
		t.Fatalf("DeleteAlertRule: %v", err)
	}
	_, err = svc.GetAlertRule(r.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
