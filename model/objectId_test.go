package model

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestObjectID(t *testing.T) {

	t.Log(primitive.NewObjectID().Hex())
}
