package service

import (
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testStreamService returns a Stream service with just enough wiring to build query criteria.
func testStreamService() *Stream {
	permissionService := NewPermission()
	streamService := NewStream()
	streamService.permissionService = &permissionService
	return &streamService
}

// TestStream_VisibilityCriteria_Owner verifies that a User sees every Stream they own
func TestStream_VisibilityCriteria_Owner(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	authorization := model.NewAuthorization()
	authorization.UserID = ownerID

	// Owners see all of their own Streams, so no additional criteria apply
	criteria := streamService.visibilityCriteria(authorization, ownerID)
	require.Equal(t, exp.Expression(exp.All()), criteria)
}

// TestStream_VisibilityCriteria_DomainOwner verifies that a domain owner sees every Stream
func TestStream_VisibilityCriteria_DomainOwner(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	authorization := model.NewAuthorization()
	authorization.UserID = primitive.NewObjectID()
	authorization.DomainOwner = true

	// Domain owners see every Stream, so no additional criteria apply
	criteria := streamService.visibilityCriteria(authorization, ownerID)
	require.Equal(t, exp.Expression(exp.All()), criteria)
}

// TestStream_VisibilityCriteria_OtherUser verifies that another User sees only the Streams their Groups allow
func TestStream_VisibilityCriteria_OtherUser(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()
	callerID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()

	authorization := model.NewAuthorization()
	authorization.UserID = callerID
	authorization.GroupIDs = id.Slice{groupID}

	before := time.Now().Unix()
	criteria := streamService.visibilityCriteria(authorization, ownerID)
	after := time.Now().Unix()

	// Another authenticated user only sees published Streams shared with their groups
	predicates := requirePublishedCriteria(t, criteria, before, after)

	permissions, ok := predicates[2].Value.(model.Permissions)
	require.True(t, ok, "defaultAllow value should be a model.Permissions slice")
	require.Contains(t, permissions, model.MagicGroupIDAnonymous)
	require.Contains(t, permissions, model.MagicGroupIDAuthenticated)
	require.Contains(t, permissions, callerID)
	require.Contains(t, permissions, groupID)
	require.NotContains(t, permissions, ownerID)
}

// TestStream_VisibilityCriteria_Anonymous verifies that an anonymous caller sees only public Streams
func TestStream_VisibilityCriteria_Anonymous(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	authorization := model.NewAuthorization()

	before := time.Now().Unix()
	criteria := streamService.visibilityCriteria(authorization, ownerID)
	after := time.Now().Unix()

	// Anonymous callers only see published Streams shared with the anonymous group
	predicates := requirePublishedCriteria(t, criteria, before, after)

	permissions, ok := predicates[2].Value.(model.Permissions)
	require.True(t, ok, "defaultAllow value should be a model.Permissions slice")
	require.Equal(t, model.Permissions{model.MagicGroupIDAnonymous}, permissions)
}

// requirePublishedCriteria asserts that criteria is the three-predicate expression
// (publishDate, unpublishDate, defaultAllow) used for non-owner callers, and returns
// the predicates for further inspection.
func requirePublishedCriteria(t *testing.T, criteria exp.Expression, before int64, after int64) []exp.Predicate {

	t.Helper()

	andExpression, ok := criteria.(exp.AndExpression)
	require.True(t, ok, "criteria should be an exp.AndExpression")
	require.Len(t, andExpression, 3)

	predicates := make([]exp.Predicate, len(andExpression))

	for index, expression := range andExpression {
		predicate, ok := expression.(exp.Predicate)
		require.True(t, ok, "each sub-expression should be an exp.Predicate")
		predicates[index] = predicate
	}

	// The Stream must currently be published...
	require.Equal(t, "publishDate", predicates[0].Field)
	require.Equal(t, exp.OperatorLessThan, predicates[0].Operator)
	requireTimestampBetween(t, before, after, predicates[0].Value)

	// ...and not yet un-published...
	require.Equal(t, "unpublishDate", predicates[1].Field)
	require.Equal(t, exp.OperatorGreaterThan, predicates[1].Operator)
	requireTimestampBetween(t, before, after, predicates[1].Value)

	// ...and shared with the caller
	require.Equal(t, "defaultAllow", predicates[2].Field)
	require.Equal(t, exp.OperatorIn, predicates[2].Operator)

	return predicates
}

// requireTimestampBetween asserts that value is an int64 Unix timestamp inside [before, after].
func requireTimestampBetween(t *testing.T, before int64, after int64, value any) {

	t.Helper()

	timestamp, ok := value.(int64)
	require.True(t, ok, "timestamp should be an int64")
	require.GreaterOrEqual(t, timestamp, before)
	require.LessOrEqual(t, timestamp, after)
}
