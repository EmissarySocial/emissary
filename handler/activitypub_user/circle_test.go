package activitypub_user

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newCircleSession returns an empty in-memory session, so that a circle lookup finds nothing
func newCircleSession(t *testing.T) data.Session {

	t.Helper()

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)
	return session
}

// TestGetCircle_PrivateUser verifies that a stranger gets a 404 for a circle of a User who is not public
func TestGetCircle_PrivateUser(t *testing.T) {

	user := newTestUser(0, 0)
	user.IsPublic = false

	ctx, _ := newCollectionContext(t, "/@sarah/pub/circles/x", nil)
	ctx.SetParamNames("circleId")
	ctx.SetParamValues(primitive.NewObjectID().Hex())

	err := GetCircle(ctx, &service.Factory{}, newCircleSession(t), &user)
	require.True(t, derp.IsNotFound(err), "got: %v", err)
}

// TestGetCircle_Paging verifies that every attempt to page into a circle is refused
func TestGetCircle_Paging(t *testing.T) {

	user := newTestUser(0, 0)

	for _, query := range []string{"?page=true", "?after=abc", "?publishDate=123"} {
		ctx, _ := newCollectionContext(t, "/@sarah/pub/circles/x"+query, nil)
		ctx.SetParamNames("circleId")
		ctx.SetParamValues(primitive.NewObjectID().Hex())

		err := GetCircle(ctx, &service.Factory{}, newCircleSession(t), &user)
		require.True(t, derp.IsForbidden(err), "query %s got: %v", query, err)
	}
}

// TestGetCircle_InvalidID verifies that a circle id that cannot name a circle is a 404
func TestGetCircle_InvalidID(t *testing.T) {

	user := newTestUser(0, 0)

	for _, value := range []string{"", "not-an-id", "123", primitive.NewObjectID().Hex() + "00", primitive.NilObjectID.Hex()} {
		ctx, _ := newCollectionContext(t, "/@sarah/pub/circles/x", nil)
		ctx.SetParamNames("circleId")
		ctx.SetParamValues(value)

		err := GetCircle(ctx, &service.Factory{}, newCircleSession(t), &user)
		require.True(t, derp.IsNotFound(err), "circleId %q got: %v", value, err)
	}
}

// TestGetCircle_UnknownCircle verifies that a circle this User does not own is a 404, the same as one that does not exist
func TestGetCircle_UnknownCircle(t *testing.T) {

	user := newTestUser(0, 0)

	ctx, recorder := newCollectionContext(t, "/@sarah/pub/circles/x", nil)
	ctx.SetParamNames("circleId")
	ctx.SetParamValues(primitive.NewObjectID().Hex())

	err := GetCircle(ctx, &service.Factory{}, newCircleSession(t), &user)
	require.True(t, derp.IsNotFound(err), "got: %v", err)
	require.Empty(t, recorder.Body.String(), "nothing may be written for a circle that is not found")
}

// TestCircleJSONLD_NamesNoMembers verifies that a circle's document carries its id and owner, and nothing about who is in it
func TestCircleJSONLD_NamesNoMembers(t *testing.T) {

	const circleURL = "https://example.com/@000000000000000000000001/pub/circles/000000000000000000000002"
	const ownerURL = "https://example.com/@000000000000000000000001"

	// Read the document back as a remote server would
	encoded, err := json.Marshal(circleJSONLD(circleURL, ownerURL))
	require.NoError(t, err)

	document := make(map[string]any)
	require.NoError(t, json.Unmarshal(encoded, &document))

	require.Equal(t, map[string]any{
		vocab.AtContext:            vocab.ContextTypeActivityStreams,
		vocab.PropertyID:           circleURL,
		vocab.PropertyType:         vocab.CoreTypeCollection,
		vocab.PropertyAttributedTo: ownerURL,
	}, document, "a circle's document must hold nothing beyond these four keys")
}
