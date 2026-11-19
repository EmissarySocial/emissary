package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestAttachment_DeleteFromCategory removes one file and its record, and leaves the rest of the category alone
func TestAttachment_DeleteFromCategory(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2)

	// The service deletes the stored file too, so give it a media server over an in-memory filesystem
	originals := afero.NewMemMapFs()
	fixture.service.mediaServer = mediaserver.New(originals, afero.NewMemMapFs(), mediaserver.NewWorkingDirectory(t.TempDir(), time.Minute, 10))
	require.NoError(t, afero.WriteFile(originals, fixture.byRank[1].Hex(), []byte("file bytes"), 0600))

	require.NoError(t, fixture.service.DeleteFromCategory(fixture.session, "Stream", fixture.objectID, "photo", fixture.byRank[1], "test"))

	// The record is gone, the file is gone, and the other photo stays
	count, err := fixture.service.CountByCategory(fixture.session, "Stream", fixture.objectID, "photo")
	require.NoError(t, err)
	require.Equal(t, 1, count)

	exists, err := afero.Exists(originals, fixture.byRank[1].Hex())
	require.NoError(t, err)
	require.False(t, exists)

	remaining, err := fixture.service.QueryByCategory(fixture.session, "Stream", fixture.objectID, "photo")
	require.NoError(t, err)
	require.Equal(t, fixture.byRank[2], remaining[0].AttachmentID)
}

// TestAttachment_DeleteFromCategory_Refusals answers 400 and deletes nothing for an ID that is not one of the category's files on the object
func TestAttachment_DeleteFromCategory_Refusals(t *testing.T) {

	fixture := newReorderFixture(t, 1)
	cover := fixture.store(t, "Stream", fixture.objectID, "cover", 1)
	otherObject := primitive.NewObjectID()
	elsewhere := fixture.store(t, "Stream", otherObject, "photo", 1)

	refuse := func(name string, attachmentID primitive.ObjectID) {
		t.Helper()

		err := fixture.service.DeleteFromCategory(fixture.session, "Stream", fixture.objectID, "photo", attachmentID, "test")
		require.Error(t, err, name)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err), name)
	}

	refuse("an unknown ID", primitive.NewObjectID())
	refuse("a file in another category", cover)
	refuse("another object's file", elsewhere)
	refuse("the zero ID", primitive.NilObjectID)

	count := func(name string, objectID primitive.ObjectID, category string, expected int) {
		t.Helper()

		result, err := fixture.service.CountByCategory(fixture.session, "Stream", objectID, category)
		require.NoError(t, err, name)
		require.Equal(t, expected, result, name)
	}

	count("the photo stays", fixture.objectID, "photo", 1)
	count("the cover stays", fixture.objectID, "cover", 1)
	count("the other object's photo stays", otherObject, "photo", 1)
}
