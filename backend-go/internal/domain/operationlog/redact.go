package operationlog

import (
	"encoding/json"
	"strings"
)

// sensitiveFields are patterns that indicate a JSON key contains
// credentials, tokens, or secrets that must be redacted from audit logs.
var sensitiveFields = []string{
	"password", "secret", "token", "api_key", "api_secret",
	"credential", "access_key", "access_secret", "refresh_token",
	"jwt", "authorization", "auth_token", "private_key",
}

// RedactSensitive redacts the value of any JSON object key whose name
// (case-insensitive) contains a sensitive field pattern.
// Non-JSON content is returned as-is.
// Used by the operationlog service's Log/LogStructured methods.
// Note: the HTTP audit middleware in httpx/middleware has its own inlined
// copy for dependency reasons. If changing this logic, update both.
func RedactSensitive(content string) string {
	if content == "" {
		return content
	}
	var raw interface{}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return content
	}
	redactAny(raw)
	b, _ := json.Marshal(raw)
	return string(b)
}

func redactNested(data map[string]interface{}) {
	for k, v := range data {
		redactKey(k, data)
		redactAny(v)
	}
}

func redactAny(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		redactNested(val)
	case []interface{}:
		for _, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				redactNested(m)
			}
		}
	}
}

func redactKey(k string, data map[string]interface{}) {
	lower := strings.ToLower(k)
	for _, sf := range sensitiveFields {
		if strings.Contains(lower, sf) {
			data[k] = "***REDACTED***"
			return
		}
	}
}
