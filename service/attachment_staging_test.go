package service

import (
	"net/http"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/mediaserver"
	"github.com/benpate/rosetta/sliceof"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stage writes one Attachment under a User, in the category, created at the given time, and returns its ID
func (fixture reorderFixture) stage(t *testing.T, userID primitive.ObjectID, category string, created time.Time) primitive.ObjectID {
	t.Helper()

	attachment := model.NewAttachment("User", userID)
	attachment.Category = category
	attachment.Original = "staged.png"
	attachment.ContentType = "image/png"

	// A record with a CreateDate is saved as an update, which keeps the date, so this backdates it
	attachment.CreateDate = created.UnixMilli()
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))

	return attachment.AttachmentID
}

// exists reports whether an Attachment is still in the database and not deleted
func (fixture reorderFixture) exists(t *testing.T, attachmentID primitive.ObjectID) bool {
	t.Helper()

	count, err := fixture.service.Count(fixture.session, exp.Equal("_id", attachmentID))
	require.NoError(t, err)

	return count > 0
}

// TestAttachment_Adopt moves a staged file onto its object, keeping what the caller set and recalculating the URL
func TestAttachment_Adopt(t *testing.T) {

	fixture := newReorderFixture(t)
	userID := primitive.NewObjectID()
	stagedID := fixture.stage(t, userID, "upload-staging", time.Now())

	attachment := fixture.load(t, stagedID)
	attachment.Category = "photo"
	attachment.Rank = 3
	attachment.SetRules(0, 0, []string{"webp"})

	require.NoError(t, fixture.service.Adopt(fixture.session, &attachment, "Stream", fixture.objectID, "test"))

	stored := fixture.load(t, stagedID)
	require.Equal(t, "Stream", stored.ObjectType)
	require.Equal(t, fixture.objectID, stored.ObjectID)
	require.Equal(t, "photo", stored.Category)
	require.Equal(t, 3, stored.Rank)
	require.Equal(t, "image/png", stored.ContentType)
	require.Equal(t, sliceof.String{"webp"}, stored.Rules.Extensions)
	require.Equal(t, "/"+fixture.objectID.Hex()+"/attachments/"+stagedID.Hex(), stored.URL)

	// It is now one of the object's photos, and no longer under the User
	photos, err := fixture.service.QueryByCategory(fixture.session, "Stream", fixture.objectID, "photo")
	require.NoError(t, err)
	require.Len(t, photos, 1)

	staged, err := fixture.service.QueryByCategory(fixture.session, "User", userID, "upload-staging")
	require.NoError(t, err)
	require.Empty(t, staged)
}

// TestAttachment_Adopt_RefusesAFileOnAnObject answers 400 and moves nothing for a file that is not staged under a User
func TestAttachment_Adopt_RefusesAFileOnAnObject(t *testing.T) {

	fixture := newReorderFixture(t, 1)
	otherObject := primitive.NewObjectID()

	attachment := fixture.load(t, fixture.byRank[1])
	err := fixture.service.Adopt(fixture.session, &attachment, "Stream", otherObject, "test")

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(err))

	stored := fixture.load(t, fixture.byRank[1])
	require.Equal(t, fixture.objectID, stored.ObjectID)
}

// TestAttachment_PurgeStaged deletes the staged files older than the cutoff, with their stored files, and nothing else
func TestAttachment_PurgeStaged(t *testing.T) {

	fixture := newReorderFixture(t, 1)
	userID := primitive.NewObjectID()
	otherUser := primitive.NewObjectID()

	originals := afero.NewMemMapFs()
	fixture.service.mediaServer = mediaserver.New(originals, afero.NewMemMapFs(), mediaserver.NewWorkingDirectory(t.TempDir(), time.Minute, 10))

	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	old := fixture.stage(t, userID, "upload-staging", twoDaysAgo)
	oldElsewhere := fixture.stage(t, otherUser, "upload-staging", twoDaysAgo)
	fresh := fixture.stage(t, userID, "upload-staging", time.Now())
	otherCategory := fixture.stage(t, userID, "mastodon-media", twoDaysAgo)
	require.NoError(t, afero.WriteFile(originals, old.Hex(), []byte("file bytes"), 0600))

	count, err := fixture.service.PurgeStaged(fixture.session, "upload-staging", time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// The two old staged files are gone, record and file
	require.False(t, fixture.exists(t, old))
	require.False(t, fixture.exists(t, oldElsewhere))

	stored, err := afero.Exists(originals, old.Hex())
	require.NoError(t, err)
	require.False(t, stored)

	// A fresh one, one in another category, and a Stream's own file all stay
	require.True(t, fixture.exists(t, fresh))
	require.True(t, fixture.exists(t, otherCategory))
	require.True(t, fixture.exists(t, fixture.byRank[1]))

	// A second run finds nothing
	count, err = fixture.service.PurgeStaged(fixture.session, "upload-staging", time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
