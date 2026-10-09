package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCarpoolSigner_SameActorMatches confirms that two stacks for the same actor on one domain get
// the same signer, so their concurrent Loads share one fetch.
func TestCarpoolSigner_SameActorMatches(t *testing.T) {

	userID := primitive.NewObjectID()

	require.Equal(t,
		carpoolSigner("one.example", model.ActorTypeApplication, primitive.NilObjectID),
		carpoolSigner("one.example", model.ActorTypeApplication, primitive.NilObjectID),
	)

	require.Equal(t,
		carpoolSigner("one.example", model.ActorTypeUser, userID),
		carpoolSigner("one.example", model.ActorTypeUser, userID),
	)
}

// TestCarpoolSigner_DifferentActorsDiffer confirms that any two different signing actors get
// different signers, so no Load is shared across credentials.
func TestCarpoolSigner_DifferentActorsDiffer(t *testing.T) {

	alice := primitive.NewObjectID()
	bob := primitive.NewObjectID()
	streamID := primitive.NewObjectID()

	tests := map[string][2]string{

		// RULE: Every domain's Application actor has a nil actorID, so only the hostname tells them apart
		"application on two domains": {
			carpoolSigner("one.example", model.ActorTypeApplication, primitive.NilObjectID),
			carpoolSigner("two.example", model.ActorTypeApplication, primitive.NilObjectID),
		},

		"application and user on one domain": {
			carpoolSigner("one.example", model.ActorTypeApplication, primitive.NilObjectID),
			carpoolSigner("one.example", model.ActorTypeUser, alice),
		},

		"two users on one domain": {
			carpoolSigner("one.example", model.ActorTypeUser, alice),
			carpoolSigner("one.example", model.ActorTypeUser, bob),
		},

		// Actor types share the nil actorID too, so the type must be part of the signer
		"application and search domain on one domain": {
			carpoolSigner("one.example", model.ActorTypeApplication, primitive.NilObjectID),
			carpoolSigner("one.example", model.ActorTypeSearchDomain, primitive.NilObjectID),
		},

		// One ObjectID space covers every collection, but the type still keeps the signers apart
		"user and stream with the same id": {
			carpoolSigner("one.example", model.ActorTypeUser, streamID),
			carpoolSigner("one.example", model.ActorTypeStream, streamID),
		},
	}

	for name, signers := range tests {
		require.NotEqual(t, signers[0], signers[1], name)
	}
}

// TestCarpoolSigner_NoNUL confirms that a signer never contains NUL, which the Carpool uses to
// separate the signer from the URL in its grouping key.
func TestCarpoolSigner_NoNUL(t *testing.T) {
	require.NotContains(t, carpoolSigner("one.example", model.ActorTypeUser, primitive.NewObjectID()), "\x00")
}
