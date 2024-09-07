package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestWebhookSchema(t *testing.T) {

	s := schema.New(WebhookSchema())
	webhook := NewWebhook()

	tests := []tableTestItem{
		{"webhookId", "000000000000000000000001", nil},
		{"label", "WEBHOOK-LABEL", nil},
		{"events.0", "user:create", nil},
		{"events.1", "user:update", nil},
		{"targetUrl", "https://example.com/webhook", nil},
	}

	tableTest_Schema(t, &s, &webhook, tests)
}
