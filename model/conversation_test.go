package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestConversationSchema(t *testing.T) {

	conversation := NewConversation()
	s := schema.New(ConversationSchema())

	table := []tableTestItem{
		{"conversationId", "123456781234567812345678", nil},
		{"userId", "aaa4bbb8ddd4ddd812345678", nil},
		{"name", "THIS-IS-MY-CONVERSATION", nil},
		{"comment", "SOME KINDA COMMENT HERE", nil},
		{"stateId", "READ", nil},
		{"icon", "flower", nil},
		{"participants.0.profileUrl", "http://johnconnor.mil.profile", nil},
		{"participants.0.name", "John Connor", nil},
		{"participants.1.profileUrl", "https://sarah.sky.net/profile", nil},
		{"participants.1.name", "Sarah Connor", nil},
	}

	tableTest_Schema(t, &s, &conversation, table)
}
