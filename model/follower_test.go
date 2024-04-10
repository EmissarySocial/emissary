package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestFollowerSchema(t *testing.T) {

	follower := NewFollower()
	s := schema.New(FollowerSchema())

	table := []tableTestItem{
		{"followerId", "123456781234567812345678", nil},
		{"parentId", "876543218765432187654321", nil},
		{"type", FollowerTypeUser, nil},
		{"method", FollowerMethodActivityPub, nil},
		{"format", MimeTypeActivityPub, nil},
		{"actor.name", "ACTOR NAME", nil},
		{"data.first", "DATA FIRST", nil},
		{"expireDate", "1234", int64(1234)},
	}

	tableTest_Schema(t, &s, &follower, table)
}
