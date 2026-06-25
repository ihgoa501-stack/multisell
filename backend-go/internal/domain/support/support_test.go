package support

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &CustomerConversation{}, &ChatMessage{}, &AutoReplyTemplate{}, &BlacklistEntry{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func createTestConversation(t *testing.T, svc *Service) *CustomerConversation {
	t.Helper()
	conv, err := svc.CreateConversation(&CreateConversationInput{
		Platform:      "shopee",
		CustomerName:  "Test User",
		CustomerEmail: "test@example.com",
		Subject:       "Order Issue",
		Priority:      "high",
	})
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}
	return conv
}

func TestConversation_Create(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	if conv.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if conv.Status != "open" {
		t.Fatalf("Status=%q, want open", conv.Status)
	}
	if conv.Priority != "high" {
		t.Fatalf("Priority=%q, want high", conv.Priority)
	}
}

func TestConversation_List(t *testing.T) {
	svc := newService(t)
	_ = createTestConversation(t, svc)
	_ = createTestConversation(t, svc)

	p := common.DefaultPagination()
	items, total, err := svc.ListConversations(&p, nil)
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
}

func TestConversation_List_FilterByStatus(t *testing.T) {
	svc := newService(t)
	_ = createTestConversation(t, svc)

	// Create a closed conversation.
	conv2, _ := svc.CreateConversation(&CreateConversationInput{
		Platform:      "lazada",
		CustomerName:  "Another User",
		CustomerEmail: "another@example.com",
		Subject:       "Shipping Delay",
	})
	_, _ = svc.SendReply(conv2.ID, "We are looking into it", false)
	_ = svc.CloseConversation(conv2.ID)

	p := common.DefaultPagination()
	items, total, err := svc.ListConversations(&p, &ConversationFilter{Status: "closed"})
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if items[0].ID != conv2.ID {
		t.Fatal("wrong conversation returned")
	}
}

func TestConversation_Get(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	got, err := svc.GetConversation(conv.ID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if got.CustomerEmail != "test@example.com" {
		t.Fatalf("CustomerEmail=%q, want test@example.com", got.CustomerEmail)
	}
}

func TestConversation_Get_NotFound(t *testing.T) {
	svc := newService(t)
	if _, err := svc.GetConversation(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestConversation_Update(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	updated, err := svc.UpdateConversation(conv.ID, &UpdateConversationInput{
		Subject:  stringPtr("Updated Subject"),
		Priority: stringPtr("low"),
	})
	if err != nil {
		t.Fatalf("UpdateConversation failed: %v", err)
	}
	if updated.Subject != "Updated Subject" {
		t.Fatalf("Subject=%q, want Updated Subject", updated.Subject)
	}
	if updated.Priority != "low" {
		t.Fatalf("Priority=%q, want low", updated.Priority)
	}
}

func TestConversation_Delete(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	if err := svc.DeleteConversation(conv.ID); err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}
	if _, err := svc.GetConversation(conv.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

// ---------- Message tests ----------

func TestSendReply(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	msg, err := svc.SendReply(conv.ID, "Thank you for your message", false)
	if err != nil {
		t.Fatalf("SendReply failed: %v", err)
	}
	if msg.ID == 0 {
		t.Fatal("expected non-zero message ID")
	}
	if msg.SenderType != "agent" {
		t.Fatalf("SenderType=%q, want agent", msg.SenderType)
	}
	if msg.AutoReplied {
		t.Fatal("expected auto_replied=false")
	}

	// Conversation should be marked as pending after reply.
	conv2, _ := svc.GetConversation(conv.ID)
	if conv2.Status != "pending" {
		t.Fatalf("Status=%q, want pending after reply", conv2.Status)
	}
	if conv2.LastMessageAt == nil {
		t.Fatal("expected LastMessageAt to be set")
	}
}

func TestSendReply_AutoReplied(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	msg, err := svc.SendReply(conv.ID, "Auto reply: we received your message", true)
	if err != nil {
		t.Fatalf("SendReply failed: %v", err)
	}
	if !msg.AutoReplied {
		t.Fatal("expected auto_replied=true")
	}
}

func TestGetMessages(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	_, _ = svc.SendReply(conv.ID, "Message 1", false)
	_, _ = svc.SendReply(conv.ID, "Message 2", false)
	_, _ = svc.SendReply(conv.ID, "Message 3", false)

	msgs, err := svc.GetMessages(conv.ID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("len=%d, want 3", len(msgs))
	}
	if msgs[0].Content != "Message 1" {
		t.Fatalf("first msg=%q, want Message 1", msgs[0].Content)
	}
}

func TestCloseConversation(t *testing.T) {
	svc := newService(t)
	conv := createTestConversation(t, svc)

	if err := svc.CloseConversation(conv.ID); err != nil {
		t.Fatalf("CloseConversation failed: %v", err)
	}

	got, _ := svc.GetConversation(conv.ID)
	if got.Status != "closed" {
		t.Fatalf("Status=%q, want closed", got.Status)
	}
}

func TestCloseConversation_NotFound(t *testing.T) {
	svc := newService(t)
	if err := svc.CloseConversation(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

// ---------- Template tests ----------

func TestCreateTemplate(t *testing.T) {
	svc := newService(t)
	tmpl, err := svc.CreateTemplate(&CreateTemplateInput{
		Name:     "退款通知",
		Category: "退款",
		Content:  "您好，您的退款申请已收到，我们将在1-3个工作日内处理。",
		Platform: "shopee",
	})
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}
	if tmpl.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if !tmpl.Enabled {
		t.Fatal("expected enabled by default")
	}
}

func TestListTemplates(t *testing.T) {
	svc := newService(t)

	_, _ = svc.CreateTemplate(&CreateTemplateInput{
		Name: "退款通知", Category: "退款", Content: "退款模板", Platform: "shopee",
	})
	_, _ = svc.CreateTemplate(&CreateTemplateInput{
		Name: "物流查询", Category: "物流", Content: "物流模板", Platform: "lazada",
	})

	// All templates.
	all, err := svc.ListTemplates("", "")
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("len=%d, want 2", len(all))
	}

	// Filter by category.
	refund, err := svc.ListTemplates("退款", "")
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(refund) != 1 {
		t.Fatalf("len=%d, want 1", len(refund))
	}
}

func TestGetTemplate(t *testing.T) {
	svc := newService(t)
	created, _ := svc.CreateTemplate(&CreateTemplateInput{
		Name: "商品咨询", Category: "商品", Content: "商品咨询模板",
	})

	got, err := svc.GetTemplate(created.ID)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if got.Name != "商品咨询" {
		t.Fatalf("Name=%q, want 商品咨询", got.Name)
	}
}

func TestUpdateTemplate(t *testing.T) {
	svc := newService(t)
	created, _ := svc.CreateTemplate(&CreateTemplateInput{
		Name: "Old Name", Category: "退款", Content: "Old Content",
	})

	updated, err := svc.UpdateTemplate(created.ID, &UpdateTemplateInput{
		Name: stringPtr("New Name"),
	})
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name=%q, want New Name", updated.Name)
	}
}

func TestDeleteTemplate(t *testing.T) {
	svc := newService(t)
	created, _ := svc.CreateTemplate(&CreateTemplateInput{
		Name: "To Delete", Category: "其他", Content: "Will be deleted",
	})

	if err := svc.DeleteTemplate(created.ID); err != nil {
		t.Fatalf("DeleteTemplate failed: %v", err)
	}
	if _, err := svc.GetTemplate(created.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

// ---------- Blacklist tests ----------

func TestAddBlacklist(t *testing.T) {
	svc := newService(t)
	entry, err := svc.AddBlacklist(&CreateBlacklistInput{
		CustomerEmail: "spam@example.com",
		CustomerName:  "Spammer",
		Reason:        "Abusive behavior",
		AddedBy:       "admin",
	})
	if err != nil {
		t.Fatalf("AddBlacklist failed: %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestCheckBlacklist(t *testing.T) {
	svc := newService(t)
	_, _ = svc.AddBlacklist(&CreateBlacklistInput{
		CustomerEmail: "blocked@example.com",
		Reason:        "Fraud",
		AddedBy:       "system",
	})

	if !svc.CheckBlacklist("blocked@example.com") {
		t.Fatal("expected blocked@example.com to be blacklisted")
	}
	if svc.CheckBlacklist("clean@example.com") {
		t.Fatal("expected clean@example.com to NOT be blacklisted")
	}
}

func TestListBlacklist(t *testing.T) {
	svc := newService(t)
	_, _ = svc.AddBlacklist(&CreateBlacklistInput{
		CustomerEmail: "spam1@example.com", Reason: "Spam",
	})
	_, _ = svc.AddBlacklist(&CreateBlacklistInput{
		CustomerEmail: "spam2@example.com", Reason: "Abuse",
	})

	items, err := svc.ListBlacklist()
	if err != nil {
		t.Fatalf("ListBlacklist failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
}

func TestDeleteBlacklist(t *testing.T) {
	svc := newService(t)
	entry, _ := svc.AddBlacklist(&CreateBlacklistInput{
		CustomerEmail: "temp@example.com", Reason: "Temporary",
	})

	if err := svc.DeleteBlacklist(entry.ID); err != nil {
		t.Fatalf("DeleteBlacklist failed: %v", err)
	}
	if svc.CheckBlacklist("temp@example.com") {
		t.Fatal("expected temp@example.com to be removed from blacklist")
	}
}

// ---------- Flow test: create → reply → messages → close ----------

func TestConversationFlow(t *testing.T) {
	svc := newService(t)

	// Create conversation.
	conv := createTestConversation(t, svc)

	// Send reply.
	msg, err := svc.SendReply(conv.ID, "Hello, how can I help you?", false)
	if err != nil {
		t.Fatalf("SendReply failed: %v", err)
	}
	if msg.ConversationID != conv.ID {
		t.Fatal("message should belong to conversation")
	}

	// Get messages.
	msgs, err := svc.GetMessages(conv.ID)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len=%d, want 1", len(msgs))
	}
	if msgs[0].Content != "Hello, how can I help you?" {
		t.Fatalf("Content=%q", msgs[0].Content)
	}

	// Close conversation.
	if err := svc.CloseConversation(conv.ID); err != nil {
		t.Fatalf("CloseConversation failed: %v", err)
	}

	// Verify closed.
	got, _ := svc.GetConversation(conv.ID)
	if got.Status != "closed" {
		t.Fatalf("Status=%q, want closed", got.Status)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("Messages len=%d, want 1", len(got.Messages))
	}
}

// ---------- Helpers ----------

func stringPtr(s string) *string {
	return &s
}
