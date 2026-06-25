package notification

import (
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
