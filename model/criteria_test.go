package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCriteria(t *testing.T) {

	id0, _ := primitive.ObjectIDFromHex("000000000000000000000000")
	id1, _ := primitive.ObjectIDFromHex("000000000000000000000001")
	id2, _ := primitive.ObjectIDFromHex("000000000000000000000002")

	c := NewCriteria()

	c.Groups = map[string][]string{
		"000000000000000000000000": {"friends", "family"},
		"000000000000000000000001": {"friends", "family", "internet randos"},
		"000000000000000000000002": {"internet randos", "system administrators"},
	}

	{
		roles := c.Roles(id0)
		require.Equal(t, []string{"friends", "family"}, roles)
	}

	{
		roles := c.Roles(id0, id1)
		require.Equal(t, []string{"friends", "family", "internet randos"}, roles)
	}

	{
		roles := c.Roles(id2, id1)
		require.Equal(t, []string{"internet randos", "system administrators", "friends", "family"}, roles)
	}

}
