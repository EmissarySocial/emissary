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
		{"actorId", "http://actor.com", nil},
		{"objectId", "https://example/object", nil},
		{"summary", "THIS_IS_A_SUMMARY", nil},
		{"content", "😀", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
