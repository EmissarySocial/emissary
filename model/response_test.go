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

		{"actor.internalId", "000000000000000000000002", nil},
		{"actor.name", "ACTOR_ NAME", nil},
		{"actor.profileUrl", "ACTOR_URL", nil},

		{"origin.internalId", "000000000000000000000003", nil},
		{"origin.url", "https://example/origin", nil},
		{"origin.label", "DOC-LABEL", nil},
		{"origin.summary", "DOC-SUMMARY", nil},

		{"objectId", "000000000000000000000005", nil},
		{"object.url", "https://example/object", nil},
		{"object.label", "DOC-LABEL", nil},
		{"object.summary", "DOC-SUMMARY", nil},
		{"object.imageUrl", "DOC-IMAGEURL", nil},
		{"object.attributedTo.0.internalId", "000000000000000000000004", nil},
		{"object.attributedTo.0.name", "DOC-AUTHOR-NAME", nil},
		{"object.attributedTo.0.profileUrl", "https://example/author", nil},
		{"object.label", "OBJECT_NAME", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
