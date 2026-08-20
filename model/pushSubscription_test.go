package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

// TestPushSubscription verifies that every PushSubscription property round-trips through the schema
func TestPushSubscription(t *testing.T) {

	sub := NewPushSubscription()

	s := schema.New(PushSubscriptionSchema())

	table := []tableTestItem{
		{"pushSubscriptionId", "123412341234123412341234", nil},
		{"userId", "123456781234567812345678", nil},
		{"endpoint", "https://push.example.com/subscription/abc123", nil},
		{"p256dh", "BNNL5ZaTfK81qhXOx23-wewhigUeFb632jN6LvRWCFH1ubQr77FE", nil},
		{"auth", "tBHItJI5svbpez7KI4CCXg", nil},
		{"userAgent", "Mozilla/5.0", nil},
	}

	tableTest_Schema(t, &s, &sub, table)
}
