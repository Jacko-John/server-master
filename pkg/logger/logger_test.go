package logger

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoggerJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	Init(buf, "info", "json")

	msg := "test message"
	Info(msg, "key", "value")

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("failed to unmarshal JSON log: %v", err)
	}

	if data["msg"] != msg {
		t.Errorf("expected msg %q, got %q", msg, data["msg"])
	}

	if data["level"] != "INFO" {
		t.Errorf("expected level INFO, got %q", data["level"])
	}

	if data["key"] != "value" {
		t.Errorf("expected key value, got %q", data["key"])
	}
}
