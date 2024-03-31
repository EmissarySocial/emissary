package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAttachmentSchema(t *testing.T) {

	attachment := NewAttachment("TEMP", primitive.NewObjectID())
	s := schema.New(AttachmentSchema())

	table := []tableTestItem{
		{"attachmentId", "123456781234567812345678", nil},
		{"objectId", "876543218765432187654321", nil},
		{"objectType", "Stream", nil},
		{"original", "ORIGINAL", nil},
		{"mediaType", "image", nil},
		{"category", "CATEGORY", nil},
		{"label", "LABEL", nil},
		{"description", "DESCRIPTION", nil},
		{"url", "http://example.com", nil},
		{"rank", "1", 1},
		{"height", "100", 100},
		{"width", "200", 200},
	}

	tableTest_Schema(t, &s, &attachment, table)
}
