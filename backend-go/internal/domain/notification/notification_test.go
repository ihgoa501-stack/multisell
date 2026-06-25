package notification

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Notification{}, &AlertRule{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

// ── Notification CRUD ────────────────────────────────────────────────

func TestNotification_Create(t *testing.T) {
	svc := newService(t)

	n := &Notification{UserID: 1, AlertType: "low_stock", Title: "Stock Alert", Severity: "warning"}
	if err := svc.Create(n); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestNotification_GetByID(t *testing.T) {
	svc := newService(t)

	created := &Notification{UserID: 1, AlertType: "test", Title: "Find Me", Severity: "info"}
	svc.Create(created)

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "Find Me" {
		t.Fatalf("Title=%q, want Find Me", got.Title)
	}
}

func TestNotification_MarkAsRead(t *testing.T) {
	svc := newService(t)

	n := &Notification{UserID: 1, AlertType: "test", Title: "Unread", Severity: "info", IsRead: 0}
	svc.Create(n)

	if err := svc.MarkAsRead(n.ID); err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}

	got, _ := svc.GetByID(n.ID)
	if got.IsRead != 1 {
		t.Fatalf("IsRead=%d, want 1", got.IsRead)
	}
}

func TestNotification_MarkAllRead(t *testing.T) {
	svc := newService(t)

	svc.Create(&Notification{UserID: 1, AlertType: "a", Title: "1", Severity: "info", IsRead: 0})
	svc.Create(&Notification{UserID: 1, AlertType: "b", Title: "2", Severity: "info", IsRead: 0})
	svc.Create(&Notification{UserID: 1, AlertType: "c", Title: "3", Severity: "info", IsRead: 1})

	if err := svc.MarkAllRead(1); err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}

	isRead := 1
	items, total, _ := svc.List(ListFilter{UserID: 1, IsRead: &isRead}, 1, 20)
	if total != 3 {
		t.Fatalf("total after mark all read=%d, want 3", total)
	}
	if len(items) != 3 {
		t.Fatalf("len=%d, want 3", len(items))
	}
}

func TestNotification_Delete(t *testing.T) {
	svc := newService(t)

	n := &Notification{UserID: 1, AlertType: "test", Title: "Delete Me", Severity: "info"}
	svc.Create(n)

	if err := svc.Delete(n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.GetByID(n.ID)
	if err == nil {
		t.Fatal("expected error for deleted notification")
	}
}

func TestNotification_UnreadCount(t *testing.T) {
	svc := newService(t)

	svc.Create(&Notification{UserID: 1, AlertType: "a", Title: "1", Severity: "info", IsRead: 0})
	svc.Create(&Notification{UserID: 1, AlertType: "b", Title: "2", Severity: "info", IsRead: 0})
	svc.Create(&Notification{UserID: 1, AlertType: "c", Title: "3", Severity: "info", IsRead: 1})

	count, err := svc.UnreadCount(1)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("count=%d, want 2", count)
	}
}

func TestNotification_List_FilterBySeverity(t *testing.T) {
	svc := newService(t)

	svc.Create(&Notification{UserID: 1, AlertType: "a", Title: "info1", Severity: "info"})
	svc.Create(&Notification{UserID: 1, AlertType: "b", Title: "warn1", Severity: "warning"})
	svc.Create(&Notification{UserID: 1, AlertType: "c", Title: "info2", Severity: "info"})

	items, total, err := svc.List(ListFilter{UserID: 1, Severity: "warning"}, 1, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 || items[0].Title != "warn1" {
		t.Fatalf("expected warn1, got %+v", items)
	}
}

func TestNotification_List_FilterByIsRead(t *testing.T) {
	svc := newService(t)

	svc.Create(&Notification{UserID: 1, AlertType: "a", Title: "read", Severity: "info", IsRead: 1})
	svc.Create(&Notification{UserID: 1, AlertType: "b", Title: "unread", Severity: "info", IsRead: 0})

	unread := 0
	items, total, err := svc.List(ListFilter{UserID: 1, IsRead: &unread}, 1, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 || items[0].Title != "unread" {
		t.Fatalf("expected unread, got %+v", items)
	}
}

// ── Alert Rules ───────────────────────────────────────────────────────

func TestAlertRule_Create(t *testing.T) {
	svc := newService(t)

	cfg, _ := json.Marshal(map[string]interface{}{"threshold": 10})
	r := &AlertRule{Name: "Low Stock", AlertType: "low_stock", Enabled: 1, Config: cfg}
	if err := svc.CreateAlertRule(r); err != nil {
		t.Fatalf("CreateAlertRule: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestAlertRule_GetByID(t *testing.T) {
	svc := newService(t)

	r := &AlertRule{Name: "Test Rule", AlertType: "test_rule", Enabled: 1}
	svc.CreateAlertRule(r)

	got, err := svc.GetAlertRule(r.ID)
	if err != nil {
		t.Fatalf("GetAlertRule: %v", err)
	}
	if got.Name != "Test Rule" {
		t.Fatalf("Name=%q, want Test Rule", got.Name)
	}
}

func TestAlertRule_List(t *testing.T) {
	svc := newService(t)

	svc.CreateAlertRule(&AlertRule{Name: "Rule A", AlertType: "a_type", Enabled: 1})
	svc.CreateAlertRule(&AlertRule{Name: "Rule B", AlertType: "b_type", Enabled: 0})

	rules, err := svc.ListAlertRules()
	if err != nil {
		t.Fatalf("ListAlertRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("len=%d, want 2", len(rules))
	}
}

func TestAlertRule_Update(t *testing.T) {
	svc := newService(t)

	r := &AlertRule{Name: "Updatable", AlertType: "upd_type", Enabled: 1}
	svc.CreateAlertRule(r)

	r.Enabled = 0
	r.Description = "updated desc"
	if err := svc.UpdateAlertRule(r); err != nil {
		t.Fatalf("UpdateAlertRule: %v", err)
	}

	got, _ := svc.GetAlertRule(r.ID)
	if got.Enabled != 0 {
		t.Fatalf("Enabled=%d, want 0", got.Enabled)
	}
	if got.Description != "updated desc" {
		t.Fatalf("Description=%q, want updated desc", got.Description)
	}
}

func TestAlertRule_Delete(t *testing.T) {
	svc := newService(t)

	r := &AlertRule{Name: "Deletable", AlertType: "del_type", Enabled: 1}
	svc.CreateAlertRule(r)

	if err := svc.DeleteAlertRule(r.ID); err != nil {
		t.Fatalf("DeleteAlertRule: %v", err)
	}
	_, err := svc.GetAlertRule(r.ID)
	if err == nil {
		t.Fatal("expected error for deleted alert rule")
	}
}
