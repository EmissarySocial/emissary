package service

import (
	"testing"

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

// TestStream_VisibilityCriteria_Owner verifies that a User sees everything in their own outbox, and nothing else
func TestStream_VisibilityCriteria_Owner(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	authorization := model.NewAuthorization()
	authorization.UserID = ownerID

	// Owners skip the permission test, but not the outbox flag
	criteria := streamService.visibilityCriteria(authorization, ownerID)
	requireOutboxPredicate(t, criteria)
}

// TestStream_VisibilityCriteria_DomainOwner verifies that a domain owner sees everything in every outbox, and nothing else
func TestStream_VisibilityCriteria_DomainOwner(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	authorization := model.NewAuthorization()
	authorization.UserID = primitive.NewObjectID()
	authorization.DomainOwner = true

	// Domain owners skip the permission test, but not the outbox flag
	criteria := streamService.visibilityCriteria(authorization, ownerID)
	requireOutboxPredicate(t, criteria)
}

// TestStream_VisibilityCriteria_OtherUser verifies that another User sees only the outbox Streams their Groups allow
func TestStream_VisibilityCriteria_OtherUser(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()
	callerID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()

	authorization := model.NewAuthorization()
	authorization.UserID = callerID
	authorization.GroupIDs = id.Slice{groupID}

	criteria := streamService.visibilityCriteria(authorization, ownerID)

	// Another authenticated user only sees outbox Streams shared with their groups
	permissions := requireSharedOutboxCriteria(t, criteria)
	require.Contains(t, permissions, model.MagicGroupIDAnonymous)
	require.Contains(t, permissions, model.MagicGroupIDAuthenticated)
	require.Contains(t, permissions, callerID)
	require.Contains(t, permissions, groupID)
	require.NotContains(t, permissions, ownerID)
}

// TestStream_VisibilityCriteria_Anonymous verifies that an anonymous caller sees only public outbox Streams
func TestStream_VisibilityCriteria_Anonymous(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	authorization := model.NewAuthorization()

	criteria := streamService.visibilityCriteria(authorization, ownerID)

	// Anonymous callers only see outbox Streams shared with the anonymous group
	permissions := requireSharedOutboxCriteria(t, criteria)
	require.Equal(t, model.Permissions{model.MagicGroupIDAnonymous}, permissions)
}

// TestStream_VisibilityCriteria_NeverReadsTheDates verifies that no caller's criteria names a publish date
func TestStream_VisibilityCriteria_NeverReadsTheDates(t *testing.T) {

	streamService := testStreamService()
	ownerID := primitive.NewObjectID()

	owner := model.NewAuthorization()
	owner.UserID = ownerID

	domainOwner := model.NewAuthorization()
	domainOwner.DomainOwner = true

	for name, authorization := range map[string]model.Authorization{
		"owner":        owner,
		"domain owner": domainOwner,
		"anonymous":    model.NewAuthorization(),
	} {
		for _, predicate := range flattenPredicates(t, streamService.visibilityCriteria(authorization, ownerID)) {
			require.NotEqual(t, "publishDate", predicate.Field, name)
			require.NotEqual(t, "unpublishDate", predicate.Field, name)
		}
	}
}

// TestUserPostsCriteria verifies that a User's posts are the Streams attributed to them
func TestUserPostsCriteria(t *testing.T) {

	ownerID := primitive.NewObjectID()

	predicates := flattenPredicates(t, userPostsCriteria(ownerID))
	require.Len(t, predicates, 1)
	require.Equal(t, "attributedTo.userId", predicates[0].Field)
	require.Equal(t, exp.OperatorEqual, predicates[0].Operator)
	require.Equal(t, ownerID, predicates[0].Value)
}

// requireOutboxPredicate asserts that criteria tests the outbox flag and nothing else
func requireOutboxPredicate(t *testing.T, criteria exp.Expression) {

	t.Helper()

	predicates := flattenPredicates(t, criteria)
	require.Len(t, predicates, 1)
	require.Equal(t, "isOutbox", predicates[0].Field)
	require.Equal(t, exp.OperatorEqual, predicates[0].Operator)
	require.Equal(t, true, predicates[0].Value)
}

// requireSharedOutboxCriteria asserts that criteria tests the outbox flag and then the
// caller's permissions, and returns those permissions
func requireSharedOutboxCriteria(t *testing.T, criteria exp.Expression) model.Permissions {

	t.Helper()

	predicates := flattenPredicates(t, criteria)
	require.Len(t, predicates, 2)

	// The Stream must be in the outbox...
	require.Equal(t, "isOutbox", predicates[0].Field)
	require.Equal(t, exp.OperatorEqual, predicates[0].Operator)
	require.Equal(t, true, predicates[0].Value)

	// ...and shared with the caller
	require.Equal(t, "defaultAllow", predicates[1].Field)
	require.Equal(t, exp.OperatorIn, predicates[1].Operator)

	permissions, ok := predicates[1].Value.(model.Permissions)
	require.True(t, ok, "defaultAllow value should be a model.Permissions slice")

	return permissions
}

// flattenPredicates returns the predicates of a criteria that is one predicate, or an AND of them
func flattenPredicates(t *testing.T, criteria exp.Expression) []exp.Predicate {

	t.Helper()

	if predicate, ok := criteria.(exp.Predicate); ok {
		return []exp.Predicate{predicate}
	}

	andExpression, ok := criteria.(exp.AndExpression)
	require.True(t, ok, "criteria should be an exp.Predicate or an exp.AndExpression")

	result := make([]exp.Predicate, 0, len(andExpression))

	for _, expression := range andExpression {
		predicate, ok := expression.(exp.Predicate)
		require.True(t, ok, "each sub-expression should be an exp.Predicate")
		result = append(result, predicate)
	}

	return result
}
