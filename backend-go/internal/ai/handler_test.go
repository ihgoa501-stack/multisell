package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler_Chat(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/chat",
		bytes.NewBufferString(`{"message":"库存不足"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response data missing or not an object")
	}
	traceID, ok := data["trace_id"].(string)
	if !ok || traceID == "" {
		t.Fatal("trace_id is empty or missing")
	}
}

func TestHandler_RunAgent(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/run",
		bytes.NewBufferString(`{"agent_id":"A5","decision_point":"stock_alert"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RunAgent(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not an object")
	}
	if data["trace_id"] == "" {
		t.Fatal("trace_id is empty")
	}
}

func TestHandler_RunAgent_MissingFields(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	// Missing agent_id — ShouldBindJSON binding:"required" validation fails.
	c.Request = httptest.NewRequest(http.MethodPost, "/run",
		bytes.NewBufferString(`{"decision_point":"stock_alert"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RunAgent(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHandler_ListTraces(t *testing.T) {
	db := newTestDB(t)
	ownerID := int64(41)
	// Seed a trace so there is data to list.
	w := NewTraceWriter(db, testLogger())
	traceID, err := w.Start(&CreateTraceInput{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		UserID:        &ownerID,
	})
	if err != nil {
		t.Fatalf("seed trace: %v", err)
	}
	_ = traceID

	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("user_id", ownerID)
	c.Request = httptest.NewRequest(http.MethodGet, "/traces", nil)

	h.ListTraces(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	if total, ok := resp["total"].(float64); !ok || total < 1 {
		t.Fatalf("total = %v, want >= 1", resp["total"])
	}
}

func TestHandler_GetTrace(t *testing.T) {
	db := newTestDB(t)
	ownerID := int64(42)
	w := NewTraceWriter(db, testLogger())
	traceID, err := w.Start(&CreateTraceInput{
		AgentID:       "A1",
		DecisionPoint: "product_scout",
		UserID:        &ownerID,
	})
	if err != nil {
		t.Fatalf("seed trace: %v", err)
	}

	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("user_id", ownerID)
	c.Params = []gin.Param{{Key: "trace_id", Value: traceID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/traces/"+traceID, nil)

	h.GetTrace(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
}

func TestHandler_ListActions(t *testing.T) {
	db := newTestDB(t)
	ownerID := int64(43)
	svc := NewService(db, testLogger())
	// Seed an action.
	_, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_la_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "list test",
		ProposedBy: "agent:A6",
		UserID:     &ownerID,
	})
	if err != nil {
		t.Fatalf("seed action: %v", err)
	}

	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("user_id", ownerID)
	c.Request = httptest.NewRequest(http.MethodGet, "/actions", nil)

	h.ListActions(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	if total, ok := resp["total"].(float64); !ok || total < 1 {
		t.Fatalf("total = %v, want >= 1", resp["total"])
	}
}

func TestHandler_LegacyDetailDoesNotCrossOwnerBoundary(t *testing.T) {
	db := newTestDB(t)
	ownerID := int64(44)
	otherOwnerID := int64(45)
	w := NewTraceWriter(db, testLogger())
	traceID, err := w.Start(&CreateTraceInput{
		AgentID: "A1", DecisionPoint: "product_scout", UserID: &otherOwnerID,
	})
	if err != nil {
		t.Fatalf("seed trace: %v", err)
	}
	svc := NewService(db, testLogger())
	action, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: traceID, SourceType: "agent_run",
		AgentID: "A1", ActionType: "product_scout", Title: "other owner",
		ProposedBy: "agent:A1", UserID: &otherOwnerID,
	})
	if err != nil {
		t.Fatalf("seed action: %v", err)
	}
	h := NewHandler(svc, NewOrchestrator(db, testLogger()), nil)

	traceRecorder := httptest.NewRecorder()
	traceContext, _ := gin.CreateTestContext(traceRecorder)
	traceContext.Set("user_id", ownerID)
	traceContext.Params = []gin.Param{{Key: "trace_id", Value: traceID}}
	traceContext.Request = httptest.NewRequest(http.MethodGet, "/traces/"+traceID, nil)
	h.GetTrace(traceContext)
	if traceRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-owner trace status = %d, want 404", traceRecorder.Code)
	}

	actionRecorder := httptest.NewRecorder()
	actionContext, _ := gin.CreateTestContext(actionRecorder)
	actionContext.Set("user_id", ownerID)
	actionContext.Params = []gin.Param{{Key: "id", Value: fmt.Sprint(action.ID)}}
	actionContext.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/actions/%d", action.ID), nil)
	h.GetAction(actionContext)
	if actionRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-owner action status = %d, want 404", actionRecorder.Code)
	}
}

func TestHandler_CreateAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	body := `{
		"source_table":"ai_trace","source_id":"trc_ca_1","source_type":"agent_run",
		"agent_id":"A6","action_type":"profit_check","title":"new action",
		"proposed_by":"agent:A6"
	}`

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/actions",
		bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateAction(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not an object")
	}
	if status, ok := data["status"].(string); !ok || status != "suggested" {
		t.Fatalf("status = %v, want suggested", data["status"])
	}
	if id, ok := data["id"].(float64); !ok || id < 1 {
		t.Fatalf("id = %v, want >= 1", data["id"])
	}
}

func TestHandler_ApproveAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	_, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_app_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "approve test",
		ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("seed action: %v", err)
	}

	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/actions/1/approve",
		bytes.NewBufferString("{}"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(42))
	c.Set("username", "approver")

	h.ApproveAction(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not an object")
	}
	if status, ok := data["status"].(string); !ok || status != "approved" {
		t.Fatalf("status = %v, want approved", data["status"])
	}
	// Verify user ID was captured from JWT context.
	if _, ok := data["approved_by_user_id"]; !ok {
		t.Fatal("approved_by_user_id missing from response")
	}
}

func TestHandler_RejectAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	_, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_rej_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "reject test",
		ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("seed action: %v", err)
	}

	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/actions/1/reject",
		bytes.NewBufferString(`{"reason":"not needed"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(99))
	c.Set("username", "rejecter")

	h.RejectAction(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not an object")
	}
	if status, ok := data["status"].(string); !ok || status != "rejected" {
		t.Fatalf("status = %v, want rejected", data["status"])
	}
}

func TestHandler_ExecuteAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	_, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exec_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "exec test",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval,
	})
	if err != nil {
		t.Fatalf("seed action: %v", err)
	}

	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/actions/1/execute",
		bytes.NewBufferString("{}"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(77))
	c.Set("username", "executor")

	h.ExecuteAction(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not an object")
	}
	if status, ok := data["status"].(string); !ok || status != "executed" {
		t.Fatalf("status = %v, want executed", data["status"])
	}
}

func TestHandler_Roster(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/agents", nil)

	h.Roster(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatal("data missing or not an array")
	}
	if len(data) == 0 {
		t.Fatal("roster is empty")
	}
}

func TestHandler_ActionNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("user_id", int64(1))
	c.Params = []gin.Param{{Key: "id", Value: "99999"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/actions/99999", nil)

	h.GetAction(c)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHandler_TraceNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("user_id", int64(1))
	c.Params = []gin.Param{{Key: "trace_id", Value: "trc_nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/traces/trc_nonexistent", nil)

	h.GetTrace(c)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHandler_AgentSpecs(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/specs", nil)

	h.AgentSpecs(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatal("data missing or not an array")
	}
	if len(data) == 0 {
		t.Fatal("agent specs array is empty")
	}
}

func TestHandler_ReviewAction(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_rev_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "review test",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval,
	})
	if err != nil {
		t.Fatalf("seed action: %v", err)
	}
	// Execute first (review requires status="executed" or "failed").
	_, err = svc.ExecuteAction(a.ID, nil, "system", "")
	if err != nil {
		t.Fatalf("execute for review: %v", err)
	}

	orch := NewOrchestrator(db, testLogger())
	h := NewHandler(svc, orch, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/actions/1/review", nil)

	h.ReviewAction(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if code, ok := resp["code"].(float64); !ok || code != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data missing or not an object")
	}
	if status, ok := data["status"].(string); !ok || status != "reviewed" {
		t.Fatalf("status = %v, want reviewed", status)
	}
}
