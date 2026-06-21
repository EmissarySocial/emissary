package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

func TestCollectionSchema(t *testing.T) {

	collection := NewCollection()
	s := schema.New(CollectionSchema())

	table := []tableTestItem{
		{"collectionId", "123456781234567812345678", nil},
		{"userId", "aaa4bbb8ddd4ddd812345678", nil},
		{"name", "THIS-IS-MY-CONVERSATION", nil},
		{"to.0", "https://johnconnor.mil/@john", nil},
		{"to.1", "https://sarah.sky.net/@sarah", nil},
		{"cc.0", "https://kyle.mil/@reese", nil},
	}

	tableTest_Schema(t, &s, &collection, table)
}

func TestCollection_HasParticipant(t *testing.T) {

	const alice = "https://alice.test/@alice"
	const bob = "https://bob.test/@bob"
	const carol = "https://carol.test/@carol"

	collection := NewCollection()
	collection.To = sliceof.String{alice}
	collection.Cc = sliceof.String{bob}

	// Actors named in "to" or "cc" are participants
	require.True(t, collection.HasParticipant(alice))
	require.True(t, collection.HasParticipant(bob))

	// An actor named in neither list is not a participant
	require.False(t, collection.HasParticipant(carol))

	// An empty actor is not a participant of a non-public collection
	require.False(t, collection.HasParticipant(""))
}

func TestCollection_HasParticipant_Public(t *testing.T) {

	const stranger = "https://stranger.test/@nobody"

	// Each of the three public-addressing spellings makes the collection readable by anyone,
	// including an actor not named in the lists and an empty/unauthenticated actor.
	for _, public := range []string{vocab.NamespaceActivityStreamsPublic, vocab.NamespaceASPublic, vocab.NamespacePublic} {

		collection := NewCollection()
		collection.Cc = sliceof.String{public}

		require.True(t, collection.HasParticipant(stranger), "public token %q should allow any actor", public)
		require.True(t, collection.HasParticipant(""), "public token %q should allow an empty actor", public)
	}
}
