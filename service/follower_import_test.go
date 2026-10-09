package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Follower UndoImport
 *
 * UndoImport is the only caller of
 * HardDeleteByID, so it is where a wrong
 * ownership field is actually felt: a rollback
 * that deletes nothing leaves IMPORT-PENDING
 * Followers behind and still reports success.
 ******************************************/

// TestFollowerUndoImport_RemovesTheImportedRecord walks the live rollback path end to end
func TestFollowerUndoImport_RemovesTheImportedRecord(t *testing.T) {

	userID := primitive.NewObjectID()

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = userID
	follower.StateID = model.FollowerStateImportPending

	service, session := newFollowerService(follower)

	importItem := model.NewImportItem()
	importItem.UserID = userID
	importItem.LocalID = follower.FollowerID

	require.Nil(t, service.UndoImport(session, &importItem))
	require.Empty(t, session.collection.records)
}

// TestFollowerUndoImport_IsScopedToTheOwner confirms a rollback cannot reach a Follower that
// belongs to a different User
func TestFollowerUndoImport_IsScopedToTheOwner(t *testing.T) {

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = primitive.NewObjectID()

	service, session := newFollowerService(follower)

	importItem := model.NewImportItem()
	importItem.UserID = primitive.NewObjectID()
	importItem.LocalID = follower.FollowerID

	require.Nil(t, service.UndoImport(session, &importItem))
	require.Len(t, session.collection.records, 1)
}

// TestFollowerUndoImport_ReportsDatabaseFailure confirms a failed rollback surfaces an error,
// so the import machinery is not told the record is gone when it is still there
func TestFollowerUndoImport_ReportsDatabaseFailure(t *testing.T) {

	userID := primitive.NewObjectID()

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = userID

	service, session := newFollowerService(follower)
	session.collection.hardDeleteError = derp.Internal("test", "database is on fire")

	importItem := model.NewImportItem()
	importItem.UserID = userID
	importItem.LocalID = follower.FollowerID

	require.NotNil(t, service.UndoImport(session, &importItem))
	require.Len(t, session.collection.records, 1)
}
