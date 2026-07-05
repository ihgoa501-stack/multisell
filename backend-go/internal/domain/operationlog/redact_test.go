package operationlog

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestRedactSensitive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expectSecret bool // whether we expect ***REDACTED*** to appear
	}{
		{name: "empty", input: "", expectSecret: false},
		{name: "non-json plain text", input: "just some text", expectSecret: false},
		{name: "password in JSON", input: `{"password":"secret123","username":"alice"}`, expectSecret: true},
		{name: "token in JSON", input: `{"token":"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0","name":"test"}`, expectSecret: true},
		{name: "api_key in JSON", input: `{"api_key":"sk-abc123","env":"prod"}`, expectSecret: true},
		{name: "case insensitive", input: `{"PASSWORD":"hunter2","Api_Secret":"xyz"}`, expectSecret: true},
		{name: "nested object", input: `{"user":{"password":"secret","name":"alice"},"config":{"key":"val"}}`, expectSecret: true},
		{name: "nested array", input: `{"items":[{"secret":"hidden"},{"name":"ok"}]}`, expectSecret: true},
		{name: "no sensitive fields", input: `{"name":"bob","age":30}`, expectSecret: false},
		{name: "authorization key", input: `{"authorization":"Bearer tok123"}`, expectSecret: true},
		{name: "jwt key", input: `{"jwt":"header.payload.sig"}`, expectSecret: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactSensitive(tc.input)
			hasMarker := strings.Contains(got, "***REDACTED***")

			if tc.expectSecret && !hasMarker {
				t.Errorf("expected ***REDACTED*** in output for %q, got %q", tc.input, got)
			}
			if !tc.expectSecret && tc.input != "" && tc.input != "just some text" {
				// For non-empty JSON input, verify it's still valid JSON.
				var parsed interface{}
				if err := json.Unmarshal([]byte(got), &parsed); err != nil {
					t.Errorf("output is not valid JSON: %v (got: %q)", err, got)
				}
			}
		})
	}
}

func TestLogStructuredRedactsContent(t *testing.T) {
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Log structured with a password in content.
	err := svc.LogStructured(&StructuredLogInput{
		Module:   "test",
		Action:   "login",
		Operator: "alice",
		Content:  `{"password":"hunter2","email":"alice@test.com"}`,
	})
	if err != nil {
		t.Fatalf("LogStructured: %v", err)
	}

	var logs []OperationLog
	if err := db.Find(&logs).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if strings.Contains(logs[0].Content, "hunter2") {
		t.Errorf("raw password leaked in audit log: %s", logs[0].Content)
	}
	if !strings.Contains(logs[0].Content, "***REDACTED***") {
		t.Errorf("expected redaction marker in content: %s", logs[0].Content)
	}
}

func TestLogRedactsContent(t *testing.T) {
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.Log("test", "api_call", "res-1", "system", `{"api_key":"sk-live-123","result":"ok"}`)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	var log OperationLog
	if err := db.Where("module = ?", "test").First(&log).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if strings.Contains(log.Content, "sk-live-123") {
		t.Errorf("API key leaked in audit log: %s", log.Content)
	}
	if !strings.Contains(log.Content, "***REDACTED***") {
		t.Errorf("expected redaction marker in content: %s", log.Content)
	}
}
