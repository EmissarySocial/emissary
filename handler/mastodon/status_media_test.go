package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mediaWithCategory returns an Attachment with a fresh ID in the given category
func mediaWithCategory(category string) model.Attachment {
	attachment := model.NewEmptyAttachment()
	attachment.AttachmentID = primitive.NewObjectID()
	attachment.Category = category
	return attachment
}

// unattachedLoader returns a loader that finds only the given unattached uploads
func unattachedLoader(uploads ...model.Attachment) func(primitive.ObjectID) (model.Attachment, error) {
	return func(attachmentID primitive.ObjectID) (model.Attachment, error) {
		for _, upload := range uploads {
			if upload.AttachmentID == attachmentID {
				return upload, nil
			}
		}
		return model.Attachment{}, derp.NotFound("unattachedLoader", "Not found", attachmentID.Hex())
	}
}

// attachmentIDs returns the IDs of a list of attachments, in order
func attachmentIDs(attachments []model.Attachment) []primitive.ObjectID {
	result := make([]primitive.ObjectID, len(attachments))
	for index, attachment := range attachments {
		result[index] = attachment.AttachmentID
	}
	return result
}

// TestPlanStatusMedia_AddsUploadsInOrder confirms a new status takes the caller's uploads in the order sent
func TestPlanStatusMedia_AddsUploadsInOrder(t *testing.T) {

	first, second := mediaWithCategory(mediaCategory), mediaWithCategory(mediaCategory)

	keep, remove, err := planStatusMedia(nil, []string{second.AttachmentID.Hex(), first.AttachmentID.Hex()}, unattachedLoader(first, second))

	require.NoError(t, err)
	require.Equal(t, []primitive.ObjectID{second.AttachmentID, first.AttachmentID}, attachmentIDs(keep))
	require.Empty(t, remove)
}

// TestPlanStatusMedia_Swaps confirms an edit can replace one photo with another
func TestPlanStatusMedia_Swaps(t *testing.T) {

	old, replacement := mediaWithCategory(mediaCategory), mediaWithCategory(mediaCategory)

	keep, remove, err := planStatusMedia([]model.Attachment{old}, []string{replacement.AttachmentID.Hex()}, unattachedLoader(replacement))

	require.NoError(t, err)
	require.Equal(t, []primitive.ObjectID{replacement.AttachmentID}, attachmentIDs(keep))
	require.Equal(t, []primitive.ObjectID{old.AttachmentID}, attachmentIDs(remove))
}

// TestPlanStatusMedia_AddsAndReorders confirms an edit can add a photo and reorder the ones already there
func TestPlanStatusMedia_AddsAndReorders(t *testing.T) {

	a, b, added := mediaWithCategory(mediaCategory), mediaWithCategory(mediaCategory), mediaWithCategory(mediaCategory)

	keep, remove, err := planStatusMedia([]model.Attachment{a, b}, []string{added.AttachmentID.Hex(), b.AttachmentID.Hex(), a.AttachmentID.Hex()}, unattachedLoader(added))

	require.NoError(t, err)
	require.Equal(t, []primitive.ObjectID{added.AttachmentID, b.AttachmentID, a.AttachmentID}, attachmentIDs(keep))
	require.Empty(t, remove)
}

// TestPlanStatusMedia_RemovesAll confirms an empty list takes every Mastodon upload off the status
func TestPlanStatusMedia_RemovesAll(t *testing.T) {

	a, b := mediaWithCategory(mediaCategory), mediaWithCategory(mediaCategory)

	keep, remove, err := planStatusMedia([]model.Attachment{a, b}, []string{}, unattachedLoader())

	require.NoError(t, err)
	require.Empty(t, keep)
	require.Equal(t, []primitive.ObjectID{a.AttachmentID, b.AttachmentID}, attachmentIDs(remove))
}

// TestPlanStatusMedia_RepeatedIDAttachesOnce confirms a repeated ID keeps only its first position
func TestPlanStatusMedia_RepeatedIDAttachesOnce(t *testing.T) {

	a, b := mediaWithCategory(mediaCategory), mediaWithCategory(mediaCategory)

	keep, _, err := planStatusMedia([]model.Attachment{a, b}, []string{b.AttachmentID.Hex(), a.AttachmentID.Hex(), b.AttachmentID.Hex()}, unattachedLoader())

	require.NoError(t, err)
	require.Equal(t, []primitive.ObjectID{b.AttachmentID, a.AttachmentID}, attachmentIDs(keep))
}

// TestPlanStatusMedia_RefusesOtherMedia confirms media that is not the caller's own unattached upload is refused
func TestPlanStatusMedia_RefusesOtherMedia(t *testing.T) {

	// Someone else's upload, or one already on another post, is not found by the loader
	_, _, err := planStatusMedia(nil, []string{primitive.NewObjectID().Hex()}, unattachedLoader())
	require.Error(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))

	// An upload the website staged for its own forms is not Mastodon media
	websiteUpload := mediaWithCategory(model.AttachmentCategoryStaging)
	_, _, err = planStatusMedia(nil, []string{websiteUpload.AttachmentID.Hex()}, unattachedLoader(websiteUpload))
	require.Error(t, err)

	// A malformed ID
	_, _, err = planStatusMedia(nil, []string{"not-an-id"}, unattachedLoader())
	require.Error(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))
}

// TestPlanStatusMedia_KeepsWebsiteMedia confirms media the website added can be kept or reordered, but not removed
func TestPlanStatusMedia_KeepsWebsiteMedia(t *testing.T) {

	websiteMedia, mastodonMedia := mediaWithCategory("images"), mediaWithCategory(mediaCategory)

	keep, remove, err := planStatusMedia([]model.Attachment{websiteMedia, mastodonMedia}, []string{mastodonMedia.AttachmentID.Hex(), websiteMedia.AttachmentID.Hex()}, unattachedLoader())
	require.NoError(t, err)
	require.Equal(t, []primitive.ObjectID{mastodonMedia.AttachmentID, websiteMedia.AttachmentID}, attachmentIDs(keep))
	require.Empty(t, remove)

	_, _, err = planStatusMedia([]model.Attachment{websiteMedia, mastodonMedia}, []string{mastodonMedia.AttachmentID.Hex()}, unattachedLoader())
	require.Error(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))
}

// TestPlanStatusMedia_RefusesTooMany confirms a status carries at most the advertised number of attachments
func TestPlanStatusMedia_RefusesTooMany(t *testing.T) {

	uploads := make([]model.Attachment, instanceMaxMediaAttachments+1)
	mediaIDs := make([]string, len(uploads))

	for index := range uploads {
		uploads[index] = mediaWithCategory(mediaCategory)
		mediaIDs[index] = uploads[index].AttachmentID.Hex()
	}

	_, _, err := planStatusMedia(nil, mediaIDs, unattachedLoader(uploads...))
	require.Error(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))

	keep, _, err := planStatusMedia(nil, mediaIDs[:instanceMaxMediaAttachments], unattachedLoader(uploads...))
	require.NoError(t, err)
	require.Len(t, keep, instanceMaxMediaAttachments)
}
