package service

import (
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Follower Export
 *
 * A Follower is owned by `parentId` + a `type`
 * discriminator, never by a `userId` field. An
 * export that queries the wrong field matches
 * nothing and reports no error, so these tests
 * pin the criteria rather than the plumbing.
 ******************************************/

// newExportFollower returns a Follower of the given type and parent, stamped with a createDate
func newExportFollower(parentType string, parentID primitive.ObjectID, createDate int64) model.Follower {

	follower := model.NewFollower()
	follower.ParentType = parentType
	follower.ParentID = parentID
	follower.CreateDate = createDate

	return follower
}

// TestFollowerExportCollection_ReturnsTheUsersFollowers is the bug itself: querying `userId`
// silently returned nothing, so a migrating User carried no followers to their new home.
func TestFollowerExportCollection_ReturnsTheUsersFollowers(t *testing.T) {

	userID := primitive.NewObjectID()
	first := newExportFollower(model.FollowerTypeUser, userID, 100)
	second := newExportFollower(model.FollowerTypeUser, userID, 200)

	service, session := newFollowerService(first, second)

	result, err := service.ExportCollection(session, userID)

	require.Nil(t, err)
	require.Equal(t, []model.IDOnly{{ID: first.FollowerID}, {ID: second.FollowerID}}, result)
}

// TestFollowerExportCollection_MatchesRangeFollowers holds the export to the same population
// the delivery path walks, which is the verification the bug report asks for.
func TestFollowerExportCollection_MatchesRangeFollowers(t *testing.T) {

	userID := primitive.NewObjectID()

	service, session := newFollowerService(
		newExportFollower(model.FollowerTypeUser, userID, 100),
		newExportFollower(model.FollowerTypeUser, userID, 200),
		newExportFollower(model.FollowerTypeUser, userID, 300),
	)

	exported, err := service.ExportCollection(session, userID)
	require.Nil(t, err)

	delivered := make([]primitive.ObjectID, 0)

	for follower := range service.RangeFollowers(session, model.FollowerTypeUser, userID) {
		delivered = append(delivered, follower.FollowerID)
	}

	require.Equal(t, delivered, model.GetIDOnly(exported))
}

// TestFollowerExportCollection_ExcludesOtherParentTypes proves the `type` clause earns its
// keep: a Stream or Search Follower can carry the same parentId and must not be exported.
func TestFollowerExportCollection_ExcludesOtherParentTypes(t *testing.T) {

	parentID := primitive.NewObjectID()
	mine := newExportFollower(model.FollowerTypeUser, parentID, 100)

	service, session := newFollowerService(
		mine,
		newExportFollower(model.FollowerTypeStream, parentID, 200),
		newExportFollower(model.FollowerTypeSearch, parentID, 300),
		newExportFollower(model.FollowerTypeSearchDomain, parentID, 400),
	)

	result, err := service.ExportCollection(session, parentID)

	require.Nil(t, err)
	require.Equal(t, []model.IDOnly{{ID: mine.FollowerID}}, result)
}

// TestFollowerExportCollection_ExcludesOtherUsers confirms one User's export cannot reach
// another User's followers
func TestFollowerExportCollection_ExcludesOtherUsers(t *testing.T) {

	userID := primitive.NewObjectID()
	mine := newExportFollower(model.FollowerTypeUser, userID, 100)

	service, session := newFollowerService(
		mine,
		newExportFollower(model.FollowerTypeUser, primitive.NewObjectID(), 200),
	)

	result, err := service.ExportCollection(session, userID)

	require.Nil(t, err)
	require.Equal(t, []model.IDOnly{{ID: mine.FollowerID}}, result)
}

// TestFollowerExportCollection_ExcludesDeletedFollowers guards the notDeleted() wrapper
func TestFollowerExportCollection_ExcludesDeletedFollowers(t *testing.T) {

	userID := primitive.NewObjectID()
	living := newExportFollower(model.FollowerTypeUser, userID, 100)
	departed := newExportFollower(model.FollowerTypeUser, userID, 200)
	departed.DeleteDate = 1

	service, session := newFollowerService(living, departed)

	result, err := service.ExportCollection(session, userID)

	require.Nil(t, err)
	require.Equal(t, []model.IDOnly{{ID: living.FollowerID}}, result)
}

// TestFollowerExportCollection_SortsByCreateDate pins the export order, so that an import
// replays a User's followers in the order they arrived
func TestFollowerExportCollection_SortsByCreateDate(t *testing.T) {

	userID := primitive.NewObjectID()
	oldest := newExportFollower(model.FollowerTypeUser, userID, 100)
	middle := newExportFollower(model.FollowerTypeUser, userID, 200)
	newest := newExportFollower(model.FollowerTypeUser, userID, 300)

	// Insert out of order, so that insertion order cannot pass for sort order
	service, session := newFollowerService(newest, oldest, middle)

	result, err := service.ExportCollection(session, userID)

	require.Nil(t, err)
	require.Equal(t, []model.IDOnly{{ID: oldest.FollowerID}, {ID: middle.FollowerID}, {ID: newest.FollowerID}}, result)
}

// TestFollowerExportCollection_EmptyIsNotAnError records that a User with no followers
// exports an empty list, rather than failing
func TestFollowerExportCollection_EmptyIsNotAnError(t *testing.T) {

	// An empty result is also exactly what the bug looked like, which is why no
	// other test in this file may rest on an empty result alone.
	service, session := newFollowerService()

	result, err := service.ExportCollection(session, primitive.NewObjectID())

	require.Nil(t, err)
	require.Empty(t, result)
}

// TestFollowerExportCollection_ZeroUserIDDoesNotMatchEverything checks the outlandish input:
// a NilObjectID is a real parentId for domain-wide Search followers, and must not leak.
func TestFollowerExportCollection_ZeroUserIDDoesNotMatchEverything(t *testing.T) {

	service, session := newFollowerService(
		newExportFollower(model.FollowerTypeSearchDomain, primitive.NilObjectID, 100),
		newExportFollower(model.FollowerTypeUser, primitive.NewObjectID(), 200),
	)

	result, err := service.ExportCollection(session, primitive.NilObjectID)

	require.Nil(t, err)
	require.Empty(t, result)
}

/******************************************
 * Follower ExportDocument
 ******************************************/

// TestFollowerExportDocument_ReturnsTheFollower verifies the document half of the export
func TestFollowerExportDocument_ReturnsTheFollower(t *testing.T) {

	userID := primitive.NewObjectID()
	follower := newExportFollower(model.FollowerTypeUser, userID, 100)
	follower.Actor.Name = "Example Follower"

	service, session := newFollowerService(follower)

	document, err := service.ExportDocument(session, userID, follower.FollowerID)
	require.Nil(t, err)

	result := model.NewFollower()
	require.Nil(t, json.Unmarshal([]byte(document), &result))

	require.Equal(t, follower.FollowerID, result.FollowerID)
	require.Equal(t, model.FollowerTypeUser, result.ParentType)
	require.Equal(t, userID, result.ParentID)
	require.Equal(t, "Example Follower", result.Actor.Name)
}

// TestFollowerExportDocument_RefusesAnotherUsersFollower keeps the per-document read scoped
// to the owner, so a valid FollowerID alone is not enough to export it
func TestFollowerExportDocument_RefusesAnotherUsersFollower(t *testing.T) {

	// ExportDocument's one uncovered branch is its json.Marshal failure, which a Follower
	// cannot reach -- the struct holds no channels, funcs, or cycles for the encoder to reject.
	follower := newExportFollower(model.FollowerTypeUser, primitive.NewObjectID(), 100)

	service, session := newFollowerService(follower)

	document, err := service.ExportDocument(session, primitive.NewObjectID(), follower.FollowerID)

	require.NotNil(t, err)
	require.Empty(t, document)
}
