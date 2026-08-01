package jobtypes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookDelivery_SendsPOST(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"url":             server.URL,
		"body":            `{"hello":"world"}`,
		"timeout_seconds": 5,
	})

	err := WebhookDelivery(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody != `{"hello":"world"}` {
		t.Errorf("expected body to be sent, got %q", receivedBody)
	}
}

func TestWebhookDelivery_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	payload, _ := json.Marshal(map[string]interface{}{
		"url":  server.URL,
		"body": `{}`,
	})

	err := WebhookDelivery(context.Background(), payload)
	if err == nil {
		t.Error("expected error for non-2xx response")
	}
}
