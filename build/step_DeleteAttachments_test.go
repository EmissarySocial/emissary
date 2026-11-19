package build

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubDeleteBuilder adds the query parameter the delete step reads
type stubDeleteBuilder struct {
	stubAttachmentBuilder
}

// QueryParam returns a parameter of the request's URL. Implements the Builder interface.
func (builder stubDeleteBuilder) QueryParam(name string) string {
	return builder._request.URL.Query().Get(name)
}

// postDelete runs the step against a POST whose query string carries the given attachmentId, when one is given.
// The Attachment service has no media server here, so these tests cover only the paths that delete nothing.
func (fixture attachmentFixture) postDelete(step StepDeleteAttachments, attachmentID string) PipelineResult {

	target := "/000000000000000000000000/delete-photo"

	if attachmentID != "" {
		target += "?attachmentId=" + attachmentID
	}

	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(""))

	builder := stubDeleteBuilder{stubAttachmentBuilder{
		_request:  request,
		_session:  fixture.session,
		_factory:  stubAttachmentFactory{attachment: fixture.service},
		_objectID: fixture.objectID,
	}}

	return applyBehavior(step.Post(builder, nil))
}

// photoCount returns how many "photo" Attachments the fixture's Stream holds
func (fixture attachmentFixture) photoCount(t *testing.T) int {
	t.Helper()

	count, err := fixture.service.CountByCategory(fixture.session, "Stream", fixture.objectID, "photo")
	require.NoError(t, err)

	return count
}

// TestStepDeleteAttachments_InvalidAttachmentIDHalts refuses a bad ID instead of deleting the whole category
func TestStepDeleteAttachments_InvalidAttachmentIDHalts(t *testing.T) {

	fixture := newAttachmentFixture(t, 1, 2)

	for _, attachmentID := range []string{"garbage", "new:0", "0123456789abcdef0123456", "%20"} {
		result := fixture.postDelete(StepDeleteAttachments{Category: "photo"}, attachmentID)

		require.NotNil(t, result.Error, attachmentID)
		require.True(t, result.Halt, attachmentID)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), attachmentID)
		require.Equal(t, 2, fixture.photoCount(t), attachmentID)
	}
}

// TestStepDeleteAttachments_MinimumRefusesTheLastFile halts when deleting one file would leave too few
func TestStepDeleteAttachments_MinimumRefusesTheLastFile(t *testing.T) {

	fixture := newAttachmentFixture(t, 1)
	result := fixture.postDelete(StepDeleteAttachments{Category: "photo", Minimum: 1}, fixture.hex(1))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
	require.Equal(t, "At least 1 file is required", derp.Message(result.Error))
	require.Equal(t, 1, fixture.photoCount(t))
}

// TestStepDeleteAttachments_MinimumRefusesTheWholeCategory halts when no ID narrows a delete that would empty the category
func TestStepDeleteAttachments_MinimumRefusesTheWholeCategory(t *testing.T) {

	fixture := newAttachmentFixture(t, 1, 2, 3)
	result := fixture.postDelete(StepDeleteAttachments{Category: "photo", Minimum: 2}, "")

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, "At least 2 files are required", derp.Message(result.Error))
	require.Equal(t, 3, fixture.photoCount(t))
}

// TestStepDeleteAttachments_MinimumCountsOnlyWhatWouldGo proceeds when the delete matches nothing the category holds
func TestStepDeleteAttachments_MinimumCountsOnlyWhatWouldGo(t *testing.T) {

	fixture := newAttachmentFixture(t, 1)
	cover := fixture.store(t, "cover", 1, "")

	// A cover's ID in the photo category matches nothing, so the photo stays and the step continues
	result := fixture.postDelete(StepDeleteAttachments{Category: "photo", Minimum: 1}, cover.Hex())

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Equal(t, "true", result.Events["attachments-updated"])
	require.Equal(t, 1, fixture.photoCount(t))
}

// TestStepDeleteAttachments_NoCriteriaDoesNothing continues without an event when nothing selects an attachment
func TestStepDeleteAttachments_NoCriteriaDoesNothing(t *testing.T) {

	fixture := newAttachmentFixture(t, 1)
	result := fixture.postDelete(StepDeleteAttachments{}, "")

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, result.Events)
	require.Equal(t, 1, fixture.photoCount(t))
}

// TestStepDeleteAttachments_Get does nothing
func TestStepDeleteAttachments_Get(t *testing.T) {
	require.Nil(t, StepDeleteAttachments{}.Get(nil, nil))
}

// TestRemainingAfterDelete counts what the category would keep
func TestRemainingAfterDelete(t *testing.T) {

	fixture := newAttachmentFixture(t, 1, 2, 3)
	cover := fixture.store(t, "cover", 1, "")

	builder := stubDeleteBuilder{stubAttachmentBuilder{
		_session:  fixture.session,
		_factory:  stubAttachmentFactory{attachment: fixture.service},
		_objectID: fixture.objectID,
	}}

	check := func(name string, category string, criteria exp.Expression, expected int) {
		t.Helper()
		remaining, err := remainingAfterDelete(builder, "Stream", fixture.objectID, category, criteria)
		require.NoError(t, err, name)
		require.Equal(t, expected, remaining, name)
	}

	photo := exp.Equal("category", "photo")

	check("one photo", "photo", photo.AndEqual("_id", fixture.byRank[2]), 2)
	check("every photo", "photo", photo, 0)
	check("an unknown id", "photo", photo.AndEqual("_id", primitive.NewObjectID()), 3)
	check("a cover's id in the photo category", "photo", photo.AndEqual("_id", cover), 3)
	check("everything on the object", "", exp.All(), 0)
	check("one of everything", "", exp.Equal("_id", cover), 3)
}
