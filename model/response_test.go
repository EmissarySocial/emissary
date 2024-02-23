package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
)

func TestResponse(t *testing.T) {

	s := schema.New(ResponseSchema())
	response := NewResponse()

	tests := []tableTestItem{
		{"responseId", "000000000000000000000001", nil},
		{"userId", "000000000000000000000001", nil},
		{"type", vocab.ActivityTypeAnnounce, nil},
		{"actor", "http://actor.com", nil},
		{"object", "https://example/object", nil},
		{"summary", "THIS_IS_A_SUMMARY", nil},
		{"content", "😀", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
