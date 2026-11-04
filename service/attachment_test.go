package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestSortByRank orders by rank, and equal ranks by ID, which is the order the files were minted in
func TestSortByRank(t *testing.T) {

	// IDs minted in sequence increase, so each later Attachment sorts after an earlier one of the same rank
	first := model.NewAttachment("Stream", primitive.NilObjectID)
	second := model.NewAttachment("Stream", primitive.NilObjectID)
	third := model.NewAttachment("Stream", primitive.NilObjectID)
	fourth := model.NewAttachment("Stream", primitive.NilObjectID)

	first.Rank = 2
	second.Rank = 0
	third.Rank = 1
	fourth.Rank = 0

	sorted := sortByRank([]model.Attachment{first, second, third, fourth})

	require.Equal(t, []primitive.ObjectID{second.AttachmentID, fourth.AttachmentID, third.AttachmentID, first.AttachmentID}, []primitive.ObjectID{sorted[0].AttachmentID, sorted[1].AttachmentID, sorted[2].AttachmentID, sorted[3].AttachmentID})

	// Nothing and one thing come back as they are
	require.Empty(t, sortByRank(nil))
	require.Empty(t, sortByRank([]model.Attachment{}))
	require.Equal(t, first.AttachmentID, sortByRank([]model.Attachment{first})[0].AttachmentID)
}

// TestAttachment_Counts counts a category, or every category, and the Attachments a criteria would select
func TestAttachment_Counts(t *testing.T) {

	fixture := newReorderFixture(t, 1, 2, 3)
	cover := fixture.store(t, "Stream", fixture.objectID, "cover", 1)
	fixture.store(t, "Stream", primitive.NewObjectID(), "photo", 1)

	deleted := fixture.load(t, fixture.byRank[3])
	require.NoError(t, fixture.session.Collection("Attachment").Delete(&deleted, "gone"))

	count := func(name string, category string, expected int) {
		t.Helper()
		result, err := fixture.service.CountByCategory(fixture.session, "Stream", fixture.objectID, category)
		require.NoError(t, err, name)
		require.Equal(t, expected, result, name)
	}

	count("photos, less the deleted one", "photo", 2)
	count("covers", "cover", 1)
	count("every category", "", 3)
	count("an empty category", "none", 0)

	match := func(name string, criteria exp.Expression, expected int) {
		t.Helper()
		result, err := fixture.service.CountByCriteria(fixture.session, "Stream", fixture.objectID, criteria)
		require.NoError(t, err, name)
		require.Equal(t, expected, result, name)
	}

	match("one photo", exp.Equal("category", "photo").AndEqual("_id", fixture.byRank[1]), 1)
	match("the deleted photo", exp.Equal("_id", fixture.byRank[3]), 0)
	match("a cover in the photo category", exp.Equal("category", "photo").AndEqual("_id", cover), 0)
	match("everything on this object", exp.All(), 3)
	match("another object's photo is out of scope", exp.Equal("category", "photo"), 2)
}

// TestAttachment_QueryByCategory_TiesByID returns files stored with equal ranks in the order they were stored
func TestAttachment_QueryByCategory_TiesByID(t *testing.T) {

	fixture := newReorderFixture(t)
	first := fixture.store(t, "Stream", fixture.objectID, "photo", 0)
	second := fixture.store(t, "Stream", fixture.objectID, "photo", 0)
	ranked := fixture.store(t, "Stream", fixture.objectID, "photo", 1)
	third := fixture.store(t, "Stream", fixture.objectID, "photo", 0)

	byCategory, err := fixture.service.QueryByCategory(fixture.session, "Stream", fixture.objectID, "photo")
	require.NoError(t, err)
	require.Len(t, byCategory, 4)
	require.Equal(t, []primitive.ObjectID{first, second, third, ranked}, []primitive.ObjectID{byCategory[0].AttachmentID, byCategory[1].AttachmentID, byCategory[2].AttachmentID, byCategory[3].AttachmentID})

	byObject, err := fixture.service.QueryByObjectID(fixture.session, "Stream", fixture.objectID)
	require.NoError(t, err)
	require.Len(t, byObject, 4)
	require.Equal(t, []primitive.ObjectID{first, second, third, ranked}, []primitive.ObjectID{byObject[0].AttachmentID, byObject[1].AttachmentID, byObject[2].AttachmentID, byObject[3].AttachmentID})
}
