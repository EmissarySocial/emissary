package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestTagSchema(t *testing.T) {

	s := schema.New(TagSchema())
	user := NewTag()

	tests := []tableTestItem{
		{"type", "Mention", nil},
		{"name", "@someone", nil},
		{"href", "https://someone.com/000000000000000000000003", nil},
	}

	tableTest_Schema(t, &s, &user, tests)

	//TODO: Include DefaultAllow?

}
