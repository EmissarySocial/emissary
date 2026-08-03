package service

import (
	"context"
	"testing"

	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestOutbox2_Send pins the record-less send contract (PROFILE-UPDATE-FEDERATION.md D-1):
// an activity without an actor is rejected (the send consumer signs with the actor's key,
// so an actor-less activity could never deliver), while a well-formed activity is accepted.
// The mock session carries no queue and no transaction spool, so "accepted" here means the
// postcommit gate ran without error; actual delivery is covered by the federation matrix.
func TestOutbox2_Send(t *testing.T) {

	service := Outbox2{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	// RULE: An activity without an actor is rejected
	missingActor := mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeUpdate,
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/@alice"},
	}

	require.Error(t, service.Send(session, missingActor))

	// A well-formed activity is accepted
	activity := mapof.Any{
		vocab.PropertyID:     "https://example.com/@alice#updates/cafebabe",
		vocab.PropertyType:   vocab.ActivityTypeUpdate,
		vocab.PropertyActor:  "https://example.com/@alice",
		vocab.PropertyCC:     []any{"https://example.com/@alice/pub/followers"},
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/@alice"},
	}

	require.NoError(t, service.Send(session, activity))
}
