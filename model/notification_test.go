package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestNotification(t *testing.T) {

	notification := NewNotification()

	s := schema.New(NotificationSchema())

	table := []tableTestItem{
		{"notificationId", "123412341234123412341234", nil},
		{"userId", "123456781234567812345678", nil},
		{"type", NotificationTypeMention, nil},
		{"actor.name", "ACTOR NAME", nil},
		{"actor.emailAddress", "ACTOR@EMAIL.COM", nil},
		{"actor.profileUrl", "https://actor.example/website", nil},
		{"actor.iconUrl", "https://actor.example/photo.jpg", nil},
		{"activityId", "https://remote.example/activity/123", nil},
		{"objectUrl", "https://remote.example/note/456", nil},
		{"objectSummary", "A plain-text summary of the object", nil},
		{"streamId", "123456781234567812345679", nil},
		{"inReplyTo", "https://local.example/stream/789", nil},
	}

	tableTest_Schema(t, &s, &notification, table)
}
