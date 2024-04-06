package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestPersonLink(t *testing.T) {

	s := schema.New(PersonLinkSchema())
	response := NewPersonLink()

	tests := []tableTestItem{
		{"userId", "000000000000000000000001", nil},
		{"name", "John Connor", nil},
		{"profileUrl", "https://john.connor.mil", nil},
		{"inboxUrl", "https://john.connor.mil/inbox", nil},
		{"emailAddress", "john.connor@mil", nil},
		{"iconUrl", "https://john.connor.mil/image", nil},
	}

	tableTest_Schema(t, &s, &response, tests)
}
