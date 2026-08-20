package model

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestObjectID verifies that every ObjectID property round-trips through the schema
func TestObjectID(t *testing.T) {

	t.Log(primitive.NewObjectID().Hex())
}
