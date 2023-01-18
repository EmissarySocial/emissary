package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestSignupForm(t *testing.T) {

	signupForm := NewSignupForm()

	s := schema.New(SignupFormSchema())

	table := []tableTestItem{
		{"title", "123412341234123412341234", nil},
		{"message", "123456781234567812345678", nil},
		{"groupId", "123456787182635481726354", nil},
		{"active", "true", true},
	}

	tableTest_Schema(t, &s, &signupForm, table)
}
