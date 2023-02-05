package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestMessageSchema(t *testing.T) {

	activity := NewMessage()
	s := schema.New(MessageSchema())

	table := []tableTestItem{
		{"activityId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"document.label", "DOCUMENT LABEL", nil},
		{"document.summary", "DOCUMENT SUMMARY", nil},
		{"origin.url", "https://origin.url", nil},
		{"contentHtml", "TEST CONTENT", nil},
		{"contentJson", `{"json":true}`, nil},
		{"folderId", "123456123456123456123456", nil},
		{"readDate", "123", int64(123)},
	}

	tableTest_Schema(t, &s, &activity, table)
}
