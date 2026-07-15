package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests reuse the in-memory folderCollection/folderSession fakes defined in
// following_test.go. They cover the Folder name-collision rule that backs the
// /.validate/folder/name endpoint, which is advisory until Save enforces it too.

// newFolderService returns a Folder service backed by an in-memory set of Folders
func newFolderService(folders ...model.Folder) (*Folder, folderSession) {

	service := NewFolder()

	return &service, folderSession{collection: &folderCollection{records: folders}}
}

/******************************************
 * ValidateLabel
 ******************************************/

// A name that nobody is using is available.
func TestFolder_ValidateLabel_Unused(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newFolderService(newTestFolder(userID, "Personal", 1))

	err := service.ValidateLabel(session, userID, primitive.NewObjectID(), "Work")

	require.Nil(t, err)
}

// A name already used by another of this User's Folders is rejected.
func TestFolder_ValidateLabel_Duplicate(t *testing.T) {

	userID := primitive.NewObjectID()
	existing := newTestFolder(userID, "Personal", 1)
	service, session := newFolderService(existing)

	err := service.ValidateLabel(session, userID, primitive.NewObjectID(), "Personal")

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
}

// A Folder may keep its own name when edited -- it does not collide with itself.
func TestFolder_ValidateLabel_ExcludesItself(t *testing.T) {

	userID := primitive.NewObjectID()
	existing := newTestFolder(userID, "Personal", 1)
	service, session := newFolderService(existing)

	err := service.ValidateLabel(session, userID, existing.FolderID, "Personal")

	require.Nil(t, err)
}

// Another User's identical Folder name is not a collision.
func TestFolder_ValidateLabel_OtherUserIsNotACollision(t *testing.T) {

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()
	service, session := newFolderService(newTestFolder(otherUserID, "Personal", 1))

	err := service.ValidateLabel(session, userID, primitive.NewObjectID(), "Personal")

	require.Nil(t, err)
}

// A soft-deleted Folder does not reserve its name.
func TestFolder_ValidateLabel_DeletedIsNotACollision(t *testing.T) {

	userID := primitive.NewObjectID()
	deleted := newTestFolder(userID, "Personal", 1)
	deleted.DeleteDate = 1000
	service, session := newFolderService(deleted)

	err := service.ValidateLabel(session, userID, primitive.NewObjectID(), "Personal")

	require.Nil(t, err)
}

// An empty name is rejected.
func TestFolder_ValidateLabel_Empty(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newFolderService()

	err := service.ValidateLabel(session, userID, primitive.NewObjectID(), "")

	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
}

// Names are compared exactly: case and whitespace make a distinct Folder.
// Pins current behavior -- tighten deliberately if that is ever wrong.
func TestFolder_ValidateLabel_ComparisonIsExact(t *testing.T) {

	userID := primitive.NewObjectID()
	service, session := newFolderService(newTestFolder(userID, "Personal", 1))

	require.Nil(t, service.ValidateLabel(session, userID, primitive.NewObjectID(), "personal"))
	require.Nil(t, service.ValidateLabel(session, userID, primitive.NewObjectID(), "Personal "))
}
