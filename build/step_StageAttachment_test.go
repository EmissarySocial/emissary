package build

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubStageBuilder is a builder over the signed-in User's outbox, which is where the staging step runs
type stubStageBuilder struct {
	stubAttachmentBuilder
	_userID primitive.ObjectID
}

// QueryParam returns a parameter of the request's URL. Implements the Builder interface.
func (builder stubStageBuilder) QueryParam(name string) string {
	return builder._request.URL.Query().Get(name)
}

// IsAuthenticated reports whether a user is signed in. Implements the Builder interface.
func (builder stubStageBuilder) IsAuthenticated() bool {
	return !builder._userID.IsZero()
}

// AuthenticatedID returns the signed-in user's ID, or zero. Implements the Builder interface.
func (builder stubStageBuilder) AuthenticatedID() primitive.ObjectID {
	return builder._userID
}

// BasePath returns the outbox's own path. Implements the Builder interface.
func (builder stubStageBuilder) BasePath() string {
	return "/@" + builder._userID.Hex()
}

// actionID names the action being run. Implements the Builder interface.
func (builder stubStageBuilder) actionID() string {
	return "upload-staging"
}

// stageFixture is the upload fixture plus the signed-in user
type stageFixture struct {
	uploadFixture
	userID primitive.ObjectID
}

// newStageFixture returns a fixture with nothing staged and a media server over an in-memory filesystem
func newStageFixture(t *testing.T) stageFixture {
	t.Helper()
	return stageFixture{uploadFixture: newUploadFixture(t), userID: primitive.NewObjectID()}
}

// fieldFile is one file to post, under its field
type fieldFile struct {
	field string
	file  postedFile
}

// request builds a multipart POST to the staging action carrying each file under its field,
// beside a form field the step must ignore
func (fixture stageFixture) request(t *testing.T, query string, files ...fieldFile) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, posted := range files {
		part, err := writer.CreateFormFile(posted.field, posted.file.name)
		require.NoError(t, err)
		_, err = part.Write([]byte(posted.file.content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.WriteField("label", "ignored"))
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, fixture.path()+query, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

// path is the staging action's own URL
func (fixture stageFixture) path() string {
	return "/@" + fixture.userID.Hex() + "/upload-staging"
}

// post runs the step's POST as the given user and returns the result with what it wrote
func (fixture stageFixture) post(userID primitive.ObjectID, request *http.Request) (PipelineResult, string) {

	builder := stubStageBuilder{
		stubAttachmentBuilder: stubAttachmentBuilder{
			_request:  request,
			_session:  fixture.session,
			_factory:  stubUploadFactory{attachment: fixture.service, media: fixture.media},
			_objectID: userID,
		},
		_userID: userID,
	}

	var buffer bytes.Buffer
	result := applyBehavior(StepStageAttachment{}.Post(builder, &buffer))

	return result, buffer.String()
}

// staged returns the user's files in the staging category
func (fixture stageFixture) staged(t *testing.T, userID primitive.ObjectID) []model.Attachment {
	t.Helper()

	result, err := fixture.service.QueryByCategory(fixture.session, "User", userID, "upload-staging")
	require.NoError(t, err)

	return result
}

// store writes one file under the user in the category, straight to the database, and returns its ID
func (fixture stageFixture) store(t *testing.T, userID primitive.ObjectID, category string) primitive.ObjectID {
	t.Helper()

	attachment := model.NewAttachment("User", userID)
	attachment.Category = category
	attachment.Original = "stored.png"
	attachment.ContentType = "image/png"
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))

	return attachment.AttachmentID
}

// TestStepStageAttachment_PostStoresTheFileUnderTheUserAndDrawsItsTile files the one posted file under the signed-in user and answers with its tile alone
func TestStepStageAttachment_PostStoresTheFileUnderTheUserAndDrawsItsTile(t *testing.T) {

	fixture := newStageFixture(t)
	result, html := fixture.post(fixture.userID, fixture.request(t, "", fieldFile{"photos", png("a.png")}))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	staged := fixture.staged(t, fixture.userID)
	require.Len(t, staged, 1)
	require.Equal(t, "User", staged[0].ObjectType)
	require.Equal(t, fixture.userID, staged[0].ObjectID)
	require.Equal(t, "upload-staging", staged[0].Category)
	require.Equal(t, "a.png", staged[0].Original)
	require.Equal(t, "image/png", staged[0].ContentType)
	require.Equal(t, "/@"+fixture.userID.Hex()+"/attachments/"+staged[0].AttachmentID.Hex(), staged[0].URL)
	stored, err := afero.Exists(fixture.originals, staged[0].AttachmentID.Hex())
	require.NoError(t, err)
	require.True(t, stored)

	// The tile is its own handle and takes focus, it previews from the user's path, its X shows on hover
	// and posts back here to replace the tile alone with no confirm, and its order field takes the
	// field the file came under
	base := "/@" + fixture.userID.Hex()
	id := staged[0].AttachmentID.Hex()
	require.Equal(t, `<div class="upload-multiple-tile hover-trigger" data-value="`+id+`" data-group="image" tabindex="0" aria-label="Move a.png"><img src="`+base+`/attachments/`+id+`?width=300" class="upload-multiple-preview" alt="a.png" draggable="false"><button type="button" class="upload-multiple-remove hover-show" hx-post="`+base+`/upload-staging?attachmentId=`+id+`" hx-params="none" hx-target="closest .upload-multiple-tile" hx-swap="outerHTML" hx-push-url="false" hx-sync="closest .upload-multiple:queue all" aria-label="Delete a.png"><i class="bi bi-x-lg"></i></button><input type="hidden" name="photos.order" value="`+id+`"></div>`, html)
}

// TestStepStageAttachment_PostTakesAnyTypeUnderAnyField stages a file that is not an image, and names its order field after the field it came under
func TestStepStageAttachment_PostTakesAnyTypeUnderAnyField(t *testing.T) {

	fixture := newStageFixture(t)
	result, html := fixture.post(fixture.userID, fixture.request(t, "", fieldFile{"data.files", text("notes.txt")}))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	staged := fixture.staged(t, fixture.userID)
	require.Len(t, staged, 1)
	require.Equal(t, "text/plain", staged[0].ContentType)

	require.Contains(t, html, `data-group="text" tabindex="0" aria-label="Move notes.txt"><i class="bi bi-file-earmark upload-multiple-icon"></i><span class="upload-multiple-filename">notes.txt</span>`)
	require.Contains(t, html, `<input type="hidden" name="data.files.order" value="`+staged[0].AttachmentID.Hex()+`">`)
	require.Equal(t, 1, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
}

// TestStepStageAttachment_PostRefusesNoFileOrMoreThanOne answers 400 and stores nothing for a body with no file, with two files, or in an encoding that carries none
func TestStepStageAttachment_PostRefusesNoFileOrMoreThanOne(t *testing.T) {

	fixture := newStageFixture(t)

	refuse := func(name string, request *http.Request) {
		t.Helper()

		result, html := fixture.post(fixture.userID, request)

		require.NotNil(t, result.Error, name)
		require.True(t, result.Halt, name)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), name)
		require.Empty(t, html, name)
	}

	refuse("no file", fixture.request(t, ""))
	refuse("two files in one field", fixture.request(t, "", fieldFile{"photos", png("a.png")}, fieldFile{"photos", png("b.png")}))
	refuse("one file in each of two fields", fixture.request(t, "", fieldFile{"photos", png("a.png")}, fieldFile{"cover", png("b.png")}))

	encoded := httptest.NewRequest(http.MethodPost, fixture.path(), strings.NewReader("photos=a.png"))
	encoded.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refuse("a url-encoded body", encoded)

	require.Empty(t, fixture.staged(t, fixture.userID))
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepStageAttachment_PostRefusesASignedOutRequest answers 401 and stores nothing when no user is signed in
func TestStepStageAttachment_PostRefusesASignedOutRequest(t *testing.T) {

	fixture := newStageFixture(t)
	result, html := fixture.post(primitive.NilObjectID, fixture.request(t, "", fieldFile{"photos", png("a.png")}))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(result.Error))
	require.Empty(t, html)
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepStageAttachment_DeleteRefusals answers 400 and deletes nothing for an ID that is not one of the user's own staged files.
// The Attachment service has no media server here, so the delete that succeeds is pinned by the service's own test of DeleteFromCategory.
func TestStepStageAttachment_DeleteRefusals(t *testing.T) {

	fixture := newStageFixture(t)
	otherUser := primitive.NewObjectID()
	theirs := fixture.store(t, otherUser, "upload-staging")
	icon := fixture.store(t, fixture.userID, "icon")

	refuse := func(name string, token string) {
		t.Helper()

		result, html := fixture.post(fixture.userID, fixture.request(t, "?attachmentId="+token))

		require.NotNil(t, result.Error, name)
		require.True(t, result.Halt, name)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), name)
		require.Empty(t, html, name)
	}

	refuse("garbage", "garbage")
	refuse("an unknown ID", primitive.NewObjectID().Hex())
	refuse("another user's staged file", theirs.Hex())
	refuse("the user's own icon", icon.Hex())

	require.Len(t, fixture.staged(t, otherUser), 1)

	icons, err := fixture.service.CountByCategory(fixture.session, "User", fixture.userID, "icon")
	require.NoError(t, err)
	require.Equal(t, 1, icons)
}

// TestStepStageAttachment_GetWritesNothing continues without drawing, because the staged files are never listed
func TestStepStageAttachment_GetWritesNothing(t *testing.T) {

	var buffer bytes.Buffer
	result := applyBehavior(StepStageAttachment{}.Get(stubStageBuilder{}, &buffer))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, buffer.String())
}
