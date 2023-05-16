package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestResponse(t *testing.T) {

	s := schema.New(ResponseSchema())
	response := NewResponse()

	tests := []tableTestItem{
		{"responseId", "000000000000000000000001", nil},
		{"type", ResponseTypeMention, nil},
		{"value", "😀", nil},

		{"actor.userId", "000000000000000000000002", nil},
		{"actor.name", "ACTOR_ NAME", nil},
		{"actor.profileUrl", "ACTOR_URL", nil},

		{"message.id", "000000000000000000000005", nil},
		{"message.url", "https://example/object", nil},
		{"message.label", "DOC-LABEL", nil},
		{"message.summary", "DOC-SUMMARY", nil},
		{"message.imageUrl", "DOC-IMAGEURL", nil},
		{"message.attributedTo.0.userId", "000000000000000000000004", nil},
		{"message.attributedTo.0.name", "DOC-AUTHOR-NAME", nil},
		{"message.attributedTo.0.profileUrl", "https://example/author", nil},
		{"message.label", "OBJECT_NAME", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
