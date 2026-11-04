package build

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubEditBuilder adds what the edit-attachments step reads beyond the upload fixture's builder:
// the query string, the object's saved state, the action's own path, and the signed-in user
type stubEditBuilder struct {
	stubAttachmentBuilder
	_object data.Object
	_userID primitive.ObjectID
}

// QueryParam returns a parameter of the request's URL. Implements the Builder interface.
func (builder stubEditBuilder) QueryParam(name string) string {
	return builder._request.URL.Query().Get(name)
}

// IsAuthenticated reports whether a user is signed in. Implements the Builder interface.
func (builder stubEditBuilder) IsAuthenticated() bool {
	return !builder._userID.IsZero()
}

// AuthenticatedID returns the signed-in user's ID, or zero. Implements the Builder interface.
func (builder stubEditBuilder) AuthenticatedID() primitive.ObjectID {
	return builder._userID
}

// object returns the Stream whose files are edited. Implements the Builder interface.
func (builder stubEditBuilder) object() data.Object {
	return builder._object
}

// BasePath returns the Stream's own path. Implements the Builder interface.
func (builder stubEditBuilder) BasePath() string {
	return "/" + builder._objectID.Hex()
}

// actionID names the action being run. Implements the Builder interface.
func (builder stubEditBuilder) actionID() string {
	return "photos"
}

// stubDraftService names the StreamDraft type, whose files belong to its Stream
type stubDraftService struct {
	service.ModelService
}

// ObjectType names the kind of object being built. Implements the ModelService interface.
func (stubDraftService) ObjectType() string {
	return "StreamDraft"
}

// stubDraftBuilder is a builder over a draft
type stubDraftBuilder struct {
	stubEditBuilder
}

// service returns a model service that names the StreamDraft type. Implements the Builder interface.
func (stubDraftBuilder) service() service.ModelService {
	return stubDraftService{}
}

// editFixture is the upload fixture plus the object the step asks about, which counts as saved
// unless a test clears its CreateDate, and the user who is signed in
type editFixture struct {
	uploadFixture
	object *model.Stream
	userID primitive.ObjectID
}

// newEditFixture stores one "photo" Attachment for each rank given, on a Stream that counts as saved
func newEditFixture(t *testing.T, ranks ...int) editFixture {
	t.Helper()

	fixture := editFixture{uploadFixture: newUploadFixture(t, ranks...), userID: primitive.NewObjectID()}

	stream := model.NewStream()
	stream.StreamID = fixture.objectID
	stream.CreateDate = 1
	fixture.object = &stream

	return fixture
}

// stage writes one file under the user in the staging category, sniffed as the given type, with
// its bytes in the media server, and returns its ID
func (fixture editFixture) stage(t *testing.T, userID primitive.ObjectID, filename string, contentType string) primitive.ObjectID {
	t.Helper()

	attachment := model.NewAttachment("User", userID)
	attachment.Category = "upload-staging"
	attachment.Original = filename
	attachment.ContentType = contentType
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))
	require.NoError(t, afero.WriteFile(fixture.originals, attachment.AttachmentID.Hex(), []byte("file bytes"), 0600))

	return attachment.AttachmentID
}

// staged returns the user's files still in the staging category
func (fixture editFixture) staged(t *testing.T, userID primitive.ObjectID) []model.Attachment {
	t.Helper()

	result, err := fixture.service.QueryByCategory(fixture.session, "User", userID, "upload-staging")
	require.NoError(t, err)

	return result
}

// storeImage writes one "photo" Attachment that was sniffed as a JPEG, and returns its ID
func (fixture editFixture) storeImage(t *testing.T, rank int) primitive.ObjectID {
	t.Helper()

	attachment := model.NewAttachment("Stream", fixture.objectID)
	attachment.Category = "photo"
	attachment.Original = "beach.jpg"
	attachment.ContentType = "image/jpeg"
	attachment.Rank = rank
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))

	return attachment.AttachmentID
}

// builder returns a builder over the fixture's Stream for the request
func (fixture editFixture) builder(request *http.Request) stubEditBuilder {
	return stubEditBuilder{
		stubAttachmentBuilder: stubAttachmentBuilder{
			_request:  request,
			_session:  fixture.session,
			_factory:  stubUploadFactory{attachment: fixture.service, media: fixture.media},
			_objectID: fixture.objectID,
		},
		_object: fixture.object,
		_userID: fixture.userID,
	}
}

// path is the action's own URL
func (fixture editFixture) path() string {
	return "/" + fixture.objectID.Hex() + "/photos"
}

// get runs the step's GET and returns the result with what it wrote
func (fixture editFixture) get(step StepEditAttachments) (PipelineResult, string) {

	request := httptest.NewRequest(http.MethodGet, fixture.path(), nil)

	var buffer bytes.Buffer
	result := applyBehavior(step.Get(fixture.builder(request), &buffer))

	return result, buffer.String()
}

// post runs the step's POST against the request and returns the result with what it wrote
func (fixture editFixture) post(step StepEditAttachments, request *http.Request) (PipelineResult, string) {

	var buffer bytes.Buffer
	result := applyBehavior(step.Post(fixture.builder(request), &buffer))

	return result, buffer.String()
}

// multipartRequest builds a POST carrying the files under "photos" and the order under "photos.order"
func (fixture editFixture) multipartRequest(t *testing.T, query string, order []string, files ...postedFile) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, file := range files {
		part, err := writer.CreateFormFile("photos", file.name)
		require.NoError(t, err)
		_, err = part.Write([]byte(file.content))
		require.NoError(t, err)
	}

	for _, value := range order {
		require.NoError(t, writer.WriteField("photos.order", value))
	}

	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, fixture.path()+query, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

// formRequest builds a URL-encoded POST
func (fixture editFixture) formRequest(query string, values url.Values) *http.Request {

	request := httptest.NewRequest(http.MethodPost, fixture.path()+query, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return request
}

// requireStored asserts that the media server holds a file for each of the given Attachments
func (fixture editFixture) requireStored(t *testing.T, attachments ...model.Attachment) {
	t.Helper()

	for _, attachment := range attachments {
		exists, err := afero.Exists(fixture.originals, attachment.AttachmentID.Hex())
		require.NoError(t, err, attachment.Original)
		require.True(t, exists, attachment.Original)
	}
}

// photoEdit returns the step as qwerty-photo would configure it, with the given limits
func photoEdit(minimum int, maximum int) StepEditAttachments {
	return StepEditAttachments{Category: "photo", Fieldname: "photos", AcceptType: "image/*", Minimum: minimum, Maximum: maximum}
}

// TestStepEditAttachments_GetDrawsTheRow writes the tile row the widget expects, in rank order, with the limits on it
func TestStepEditAttachments_GetDrawsTheRow(t *testing.T) {

	fixture := newEditFixture(t, 2)
	image := fixture.storeImage(t, 1)
	plain := fixture.byRank[2]

	result, html := fixture.get(photoEdit(1, 64))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	base := "/" + fixture.objectID.Hex()

	require.True(t, strings.HasPrefix(html, `<div class="upload-multiple-tiles" data-url="`+base+`/photos" data-name="photos" data-accept="image/*" data-minimum="1" data-maximum="64" data-count="2" hx-post="`+base+`/photos" hx-trigger="upload-multiple-added, upload-multiple-sorted" hx-params="photos,photos.order" hx-target="this" hx-swap="outerHTML" hx-push-url="false" hx-sync="closest .upload-multiple:queue all">`), html)
	require.True(t, strings.HasSuffix(html, "</div></div>"), html)

	// Every tile is its own handle and takes focus, with an X that shows on hover; an image tile
	// previews the file, and any other tile shows an icon and the filename
	require.Contains(t, html, `<div class="upload-multiple-tile hover-trigger" data-value="`+image.Hex()+`" data-group="image" tabindex="0" aria-label="Move beach.jpg"><img src="`+base+`/attachments/`+image.Hex()+`?width=300" class="upload-multiple-preview" alt="beach.jpg" draggable="false"><button type="button" class="upload-multiple-remove hover-show" hx-post="`+base+`/photos?attachmentId=`+image.Hex()+`" hx-params="none" hx-confirm="Delete this file?" hx-target="closest .upload-multiple-tiles" hx-swap="outerHTML" hx-push-url="false" hx-sync="closest .upload-multiple:queue all" aria-label="Delete beach.jpg"><i class="bi bi-x-lg"></i></button><input type="hidden" name="photos.order" value="`+image.Hex()+`"></div>`)
	require.Contains(t, html, `<div class="upload-multiple-tile hover-trigger" data-value="`+plain.Hex()+`" tabindex="0" aria-label="Move file.jpg"><i class="bi bi-file-earmark upload-multiple-icon"></i><span class="upload-multiple-filename">file.jpg</span>`)
	require.Less(t, strings.Index(html, image.Hex()), strings.Index(html, plain.Hex()))
	require.Equal(t, 2, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
}

// TestStepEditAttachments_GetEscapesTheFilename draws a hostile filename as text
func TestStepEditAttachments_GetEscapesTheFilename(t *testing.T) {

	fixture := newEditFixture(t)

	attachment := model.NewAttachment("Stream", fixture.objectID)
	attachment.Category = "photo"
	attachment.Original = `<script>alert(1)</script> & "more".pdf`
	attachment.Rank = 1
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))

	_, html := fixture.get(photoEdit(0, 64))

	require.NotContains(t, html, "<script>")
	require.Contains(t, html, `<span class="upload-multiple-filename">&lt;script&gt;alert(1)&lt;/script&gt; &amp; &#34;more&#34;.pdf</span>`)
	require.Contains(t, html, `aria-label="Delete &lt;script&gt;alert(1)&lt;/script&gt; &amp; &#34;more&#34;.pdf"`)
}

// TestStepEditAttachments_GetOnANewObjectWritesNothing leaves the buffer empty for an object that is not saved yet
func TestStepEditAttachments_GetOnANewObjectWritesNothing(t *testing.T) {

	fixture := newEditFixture(t, 1)
	fixture.object.CreateDate = 0

	result, html := fixture.get(photoEdit(1, 64))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, html)
}

// TestStepEditAttachments_UploadAppendsAfterTheHighestRank stores the posted files after the category's highest rank and draws the row again
func TestStepEditAttachments_UploadAppendsAfterTheHighestRank(t *testing.T) {

	fixture := newEditFixture(t, 1, 5)
	result, html := fixture.post(photoEdit(1, 64), fixture.multipartRequest(t, "", nil, png("a.png"), png("b.png")))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	photos := fixture.photos(t)
	require.Len(t, photos, 4)
	require.Equal(t, []int{1, 5, 6, 7}, []int{photos[0].Rank, photos[1].Rank, photos[2].Rank, photos[3].Rank})
	require.Equal(t, "a.png", photos[2].Original)
	require.Equal(t, "b.png", photos[3].Original)
	require.Equal(t, "image/png", photos[3].ContentType)
	fixture.requireStored(t, photos[2:]...)

	require.Contains(t, html, `data-count="4"`)
	require.Equal(t, 4, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
	require.Less(t, strings.Index(html, photos[2].AttachmentID.Hex()), strings.Index(html, photos[3].AttachmentID.Hex()))
}

// TestStepEditAttachments_UploadOverTheMaximumStoresNothing refuses the whole batch with an inline message
func TestStepEditAttachments_UploadOverTheMaximumStoresNothing(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)
	result, html := fixture.post(photoEdit(1, 3), fixture.multipartRequest(t, "", nil, png("a.png"), png("b.png")))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.True(t, derp.IsValidationError(result.Error))
	require.Equal(t, "At most 3 files are allowed", derp.Message(result.Error))
	require.Empty(t, html)
	require.Len(t, fixture.photos(t), 2)
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepEditAttachments_UploadRefusesABadFileBeforeStoringAny halts on a file that is not an image, with the first file still unstored
func TestStepEditAttachments_UploadRefusesABadFileBeforeStoringAny(t *testing.T) {

	fixture := newEditFixture(t)
	result, html := fixture.post(photoEdit(0, 64), fixture.multipartRequest(t, "", nil, png("a.png"), text("notes.txt")))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
	require.Empty(t, html)
	require.Empty(t, fixture.photos(t))
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepEditAttachments_SaveBelowTheMinimumIsRefused refuses a create form's Save that leaves the category short, with an inline message
func TestStepEditAttachments_SaveBelowTheMinimumIsRefused(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0

	result, html := fixture.post(photoEdit(1, 64), fixture.multipartRequest(t, "", nil))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.True(t, derp.IsValidationError(result.Error))
	require.Equal(t, "At least 1 file is required", derp.Message(result.Error))
	require.Empty(t, html)
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepEditAttachments_UploadBelowTheMinimumIsStored stores one file on a saved object that is still short of its minimum, and draws the row
func TestStepEditAttachments_UploadBelowTheMinimumIsStored(t *testing.T) {

	fixture := newEditFixture(t)
	result, html := fixture.post(photoEdit(3, 64), fixture.multipartRequest(t, "", nil, png("a.png")))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Len(t, fixture.photos(t), 1)
	require.Contains(t, html, `data-minimum="3"`)
	require.Contains(t, html, `data-count="1"`)
}

// TestStepEditAttachments_SortBelowTheMinimumIsApplied re-ranks a saved object's files while the category is short of its minimum
func TestStepEditAttachments_SortBelowTheMinimumIsApplied(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)
	result, _ := fixture.post(photoEdit(3, 64), fixture.multipartRequest(t, "", []string{fixture.hex(2), fixture.hex(1)}))

	require.Nil(t, result.Error)
	require.Equal(t, 1, fixture.rankOf(t, 2))
	require.Equal(t, 2, fixture.rankOf(t, 1))
}

// TestStepEditAttachments_SortOverTheMaximumIsApplied re-ranks a saved object's files while the category holds more than its maximum, and deletes none
func TestStepEditAttachments_SortOverTheMaximumIsApplied(t *testing.T) {

	fixture := newEditFixture(t, 1, 2, 3)
	order := []string{fixture.hex(3), fixture.hex(2), fixture.hex(1)}
	result, html := fixture.post(photoEdit(1, 2), fixture.multipartRequest(t, "", order))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Equal(t, 1, fixture.rankOf(t, 3))
	require.Equal(t, 2, fixture.rankOf(t, 2))
	require.Equal(t, 3, fixture.rankOf(t, 1))
	require.Equal(t, 3, fixture.photoCount(t))
	require.Contains(t, html, `data-maximum="2" data-count="3"`)
}

// TestStepEditAttachments_UploadOverAFullCategoryIsRefused refuses one more file on a saved object already over its maximum, and stores nothing
func TestStepEditAttachments_UploadOverAFullCategoryIsRefused(t *testing.T) {

	fixture := newEditFixture(t, 1, 2, 3)
	result, _ := fixture.post(photoEdit(1, 2), fixture.multipartRequest(t, "", nil, png("a.png")))

	require.NotNil(t, result.Error)
	require.True(t, derp.IsValidationError(result.Error))
	require.Equal(t, "At most 2 files are allowed", derp.Message(result.Error))
	require.Equal(t, 3, fixture.photoCount(t))
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepEditAttachments_SaveOverTheMaximumKeepsWhatFits adopts a create form's staged files up to the maximum, in the order posted, and leaves the rest staged
func TestStepEditAttachments_SaveOverTheMaximumKeepsWhatFits(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0
	a := fixture.stage(t, fixture.userID, "a.png", "image/png")
	b := fixture.stage(t, fixture.userID, "b.png", "image/png")
	c := fixture.stage(t, fixture.userID, "c.png", "image/png")

	result, html := fixture.post(photoEdit(1, 2), fixture.multipartRequest(t, "", []string{c.Hex(), a.Hex(), b.Hex()}))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, html)

	photos := fixture.photos(t)
	require.Len(t, photos, 2)
	require.Equal(t, c, photos[0].AttachmentID)
	require.Equal(t, a, photos[1].AttachmentID)

	staged := fixture.staged(t, fixture.userID)
	require.Len(t, staged, 1)
	require.Equal(t, b, staged[0].AttachmentID)
}

// TestStepEditAttachments_SaveOverTheMaximumKeepsThePostedFilesFirst stores a create form's posted files before adopting any staged file, because they rank first
func TestStepEditAttachments_SaveOverTheMaximumKeepsThePostedFilesFirst(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0
	staged := fixture.stage(t, fixture.userID, "staged.png", "image/png")

	result, _ := fixture.post(photoEdit(1, 2), fixture.multipartRequest(t, "", []string{staged.Hex()}, png("a.png"), png("b.png"), png("c.png")))

	require.Nil(t, result.Error)

	photos := fixture.photos(t)
	require.Len(t, photos, 2)
	require.Equal(t, "a.png", photos[0].Original)
	require.Equal(t, "b.png", photos[1].Original)
	fixture.requireStored(t, photos...)
	require.Len(t, fixture.staged(t, fixture.userID), 1)
}

// TestStepEditAttachments_SaveDropsAWrongTypePastTheMaximum adopts what fits and leaves a refused type staged, when that file was past the maximum anyway
func TestStepEditAttachments_SaveDropsAWrongTypePastTheMaximum(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0
	a := fixture.stage(t, fixture.userID, "a.png", "image/png")
	notes := fixture.stage(t, fixture.userID, "notes.txt", "text/plain")

	result, _ := fixture.post(photoEdit(1, 1), fixture.multipartRequest(t, "", []string{a.Hex(), notes.Hex()}))

	require.Nil(t, result.Error)

	photos := fixture.photos(t)
	require.Len(t, photos, 1)
	require.Equal(t, a, photos[0].AttachmentID)
	require.Len(t, fixture.staged(t, fixture.userID), 1)
}

// TestStepEditAttachments_LimitSave keeps the posted files, then the staged ones, up to the maximum, and refuses a count short of the minimum
func TestStepEditAttachments_LimitSave(t *testing.T) {

	posted := func(names ...string) []*multipart.FileHeader {
		result := make([]*multipart.FileHeader, 0, len(names))
		for _, name := range names {
			result = append(result, &multipart.FileHeader{Filename: name})
		}
		return result
	}

	staged := func(names ...string) []model.Attachment {
		result := make([]model.Attachment, 0, len(names))
		for _, name := range names {
			result = append(result, model.Attachment{Original: name})
		}
		return result
	}

	keeps := func(name string, minimum int, maximum int, existing int, files []*multipart.FileHeader, stagedFiles []model.Attachment, wantFiles []string, wantStaged []string) {
		t.Helper()

		gotFiles, gotStaged, err := photoEdit(minimum, maximum).limitSave(existing, files, stagedFiles)
		require.NoError(t, err, name)

		fileNames := make([]string, 0, len(gotFiles))
		for _, file := range gotFiles {
			fileNames = append(fileNames, file.Filename)
		}

		stagedNames := make([]string, 0, len(gotStaged))
		for _, attachment := range gotStaged {
			stagedNames = append(stagedNames, attachment.Original)
		}

		require.Equal(t, wantFiles, fileNames, name)
		require.Equal(t, wantStaged, stagedNames, name)
	}

	refuses := func(name string, minimum int, maximum int, existing int, files []*multipart.FileHeader, stagedFiles []model.Attachment) {
		t.Helper()

		gotFiles, gotStaged, err := photoEdit(minimum, maximum).limitSave(existing, files, stagedFiles)
		require.True(t, derp.IsValidationError(err), name)
		require.Equal(t, minimumMessage(minimum), derp.Message(err), name)
		require.Nil(t, gotFiles, name)
		require.Nil(t, gotStaged, name)
	}

	keeps("everything fits", 1, 4, 0, posted("a", "b"), staged("c", "d"), []string{"a", "b"}, []string{"c", "d"})
	keeps("the staged tail is dropped", 1, 3, 0, posted("a", "b"), staged("c", "d"), []string{"a", "b"}, []string{"c"})
	keeps("every staged file is dropped", 1, 2, 0, posted("a", "b"), staged("c", "d"), []string{"a", "b"}, []string{})
	keeps("the posted tail is dropped", 1, 1, 0, posted("a", "b"), staged("c"), []string{"a"}, []string{})
	keeps("staged alone", 1, 2, 0, nil, staged("a", "b", "c"), []string{}, []string{"a", "b"})
	keeps("existing files take room first", 1, 3, 2, posted("a"), staged("b"), []string{"a"}, []string{})
	keeps("existing files over the maximum leave no room", 1, 2, 3, posted("a"), staged("b"), []string{}, []string{})
	keeps("nothing posted with no minimum", 0, 2, 0, nil, nil, []string{}, []string{})
	keeps("exactly the minimum", 2, 2, 0, posted("a"), staged("b"), []string{"a"}, []string{"b"})

	refuses("nothing posted", 1, 2, 0, nil, nil)
	refuses("one short", 3, 4, 0, posted("a"), staged("b"))
}

// TestStepEditAttachments_LimitChange refuses only an add that passes the maximum
func TestStepEditAttachments_LimitChange(t *testing.T) {

	passes := func(name string, minimum int, maximum int, existing int, adding int) {
		t.Helper()
		require.NoError(t, photoEdit(minimum, maximum).limitChange(existing, adding), name)
	}

	refuses := func(name string, maximum int, existing int, adding int) {
		t.Helper()

		err := photoEdit(0, maximum).limitChange(existing, adding)
		require.True(t, derp.IsValidationError(err), name)
		require.Equal(t, maximumMessage(maximum), derp.Message(err), name)
	}

	passes("an add that fits", 1, 4, 2, 1)
	passes("an add up to the maximum", 1, 4, 2, 2)
	passes("an add short of the minimum", 3, 4, 0, 1)
	passes("a sort short of the minimum", 3, 4, 1, 0)
	passes("a sort over the maximum", 1, 2, 5, 0)
	passes("a sort of nothing", 1, 2, 0, 0)

	refuses("an add past the maximum", 4, 3, 2)
	refuses("an add to a full category", 2, 2, 1)
	refuses("an add to a category already over", 2, 5, 1)
}

// TestStepEditAttachments_SortAppliesThePostedOrder re-ranks the saved files in the order posted, skips what is not a saved ID, and draws the new order
func TestStepEditAttachments_SortAppliesThePostedOrder(t *testing.T) {

	fixture := newEditFixture(t, 1, 2, 3)
	order := []string{fixture.hex(3), "garbage", fixture.hex(1), primitive.NewObjectID().Hex(), fixture.hex(2)}
	result, html := fixture.post(photoEdit(1, 64), fixture.multipartRequest(t, "", order))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Equal(t, 1, fixture.rankOf(t, 3))
	require.Equal(t, 2, fixture.rankOf(t, 1))
	require.Equal(t, 3, fixture.rankOf(t, 2))

	// The row comes back in the new order
	require.Less(t, strings.Index(html, fixture.hex(3)), strings.Index(html, fixture.hex(1)))
	require.Less(t, strings.Index(html, fixture.hex(1)), strings.Index(html, fixture.hex(2)))
	require.Equal(t, 0, fixture.storedFiles(t))
}

// TestStepEditAttachments_SortReadsTheBodyOnly ignores an order written into the query string
func TestStepEditAttachments_SortReadsTheBodyOnly(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)
	query := "?photos.order=" + fixture.hex(2) + "&photos.order=" + fixture.hex(1)
	result, _ := fixture.post(photoEdit(0, 64), fixture.formRequest(query, url.Values{}))

	require.Nil(t, result.Error)
	require.Equal(t, 1, fixture.rankOf(t, 1))
	require.Equal(t, 2, fixture.rankOf(t, 2))
}

// TestStepEditAttachments_DeleteRefusals halts and deletes nothing for an ID that is not one of the category's files, or that would leave too few
func TestStepEditAttachments_DeleteRefusals(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)
	cover := fixture.store(t, "cover", 1, "")

	refuse := func(name string, step StepEditAttachments, token string, code int) {
		t.Helper()

		result, html := fixture.post(step, fixture.multipartRequest(t, "?attachmentId="+token, nil))

		require.NotNil(t, result.Error, name)
		require.True(t, result.Halt, name)
		require.Equal(t, code, derp.ErrorCode(result.Error), name)
		require.Empty(t, html, name)
	}

	refuse("garbage", photoEdit(0, 64), "garbage", http.StatusBadRequest)
	refuse("a short hex", photoEdit(0, 64), "0123456789abcdef0123456", http.StatusBadRequest)
	refuse("an unknown ID", photoEdit(0, 64), primitive.NewObjectID().Hex(), http.StatusBadRequest)
	refuse("a cover in the photo category", photoEdit(0, 64), cover.Hex(), http.StatusBadRequest)
	refuse("the minimum", photoEdit(2, 64), fixture.hex(1), derp.ErrorCode(derp.Validation("")))

	// The refusal at the minimum carries the message the widget shows
	result, _ := fixture.post(photoEdit(2, 64), fixture.multipartRequest(t, "?attachmentId="+fixture.hex(1), nil))
	require.Equal(t, "At least 2 files are required", derp.Message(result.Error))

	require.Equal(t, 2, fixture.photoCount(t))

	covers, err := fixture.service.CountByCategory(fixture.session, "Stream", fixture.objectID, "cover")
	require.NoError(t, err)
	require.Equal(t, 1, covers)
}

// TestStepEditAttachments_PostOnANewObjectStoresAndWritesNothing stores a create form's files under the new object's ID and draws no row
func TestStepEditAttachments_PostOnANewObjectStoresAndWritesNothing(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0

	result, html := fixture.post(photoEdit(1, 64), fixture.multipartRequest(t, "", nil, png("a.png"), png("b.png")))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, html)

	photos := fixture.photos(t)
	require.Len(t, photos, 2)
	require.Equal(t, []int{1, 2}, []int{photos[0].Rank, photos[1].Rank})
	require.Equal(t, fixture.objectID, photos[0].ObjectID)
	fixture.requireStored(t, photos...)
}

// TestStepEditAttachments_UnsupportedBodyIsRefused answers 400 to a body in an encoding a form never sends
func TestStepEditAttachments_UnsupportedBodyIsRefused(t *testing.T) {

	fixture := newEditFixture(t, 1)

	request := httptest.NewRequest(http.MethodPost, fixture.path(), strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")

	result, html := fixture.post(photoEdit(0, 64), request)

	require.NotNil(t, result.Error)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
	require.Empty(t, html)
}

// TestAttachmentObjectType files a draft's attachments under its Stream
func TestAttachmentObjectType(t *testing.T) {
	require.Equal(t, "Stream", attachmentObjectType(stubEditBuilder{}))
	require.Equal(t, "Stream", attachmentObjectType(stubDraftBuilder{}))
}

// TestMaximumMessage words the limit for one file and for several
func TestMaximumMessage(t *testing.T) {
	require.Equal(t, "At most 1 file is allowed", maximumMessage(1))
	require.Equal(t, "At most 64 files are allowed", maximumMessage(64))
}

// TestStepEditAttachments_PostAdoptsTheStagedFilesTheOrderNames moves the caller's staged files onto a new object, in the order posted, with the step's category and rules, and draws nothing
func TestStepEditAttachments_PostAdoptsTheStagedFilesTheOrderNames(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0
	a := fixture.stage(t, fixture.userID, "a.png", "image/png")
	b := fixture.stage(t, fixture.userID, "b.png", "image/png")

	step := photoEdit(1, 64)
	step.RuleTypes = []string{"webp"}
	result, html := fixture.post(step, fixture.multipartRequest(t, "", []string{b.Hex(), a.Hex()}))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, html)

	// Both are the new object's photos now, ranked in the order posted, with the step's rules
	photos := fixture.photos(t)
	require.Len(t, photos, 2)
	require.Equal(t, b, photos[0].AttachmentID)
	require.Equal(t, a, photos[1].AttachmentID)
	require.Equal(t, []int{1, 2}, []int{photos[0].Rank, photos[1].Rank})
	require.Equal(t, "Stream", photos[0].ObjectType)
	require.Equal(t, fixture.objectID, photos[0].ObjectID)
	require.Equal(t, "image/png", photos[0].ContentType)
	require.Equal(t, sliceof.String{"webp"}, photos[0].Rules.Extensions)
	require.Equal(t, "/"+fixture.objectID.Hex()+"/attachments/"+b.Hex(), photos[0].URL)
	require.Empty(t, fixture.staged(t, fixture.userID))
	fixture.requireStored(t, photos...)
}

// TestStepEditAttachments_AdoptionJoinsTheSavedFiles adopts a staged file onto a saved object, ranked after its saved files, and the posted order still applies to the saved ones.
// The adopted file is not moved by the order here, because the fixture's service has no host, so its relative URL would fail the second save's validation.
func TestStepEditAttachments_AdoptionJoinsTheSavedFiles(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)
	c := fixture.stage(t, fixture.userID, "c.png", "image/png")

	order := []string{fixture.hex(2), fixture.hex(1), c.Hex()}
	result, html := fixture.post(photoEdit(1, 64), fixture.multipartRequest(t, "", order))

	require.Nil(t, result.Error)
	require.Equal(t, 1, fixture.rankOf(t, 2))
	require.Equal(t, 2, fixture.rankOf(t, 1))
	require.Equal(t, 3, fixture.rankByID(t, c))

	// The row comes back with the adopted file last, and the file is no longer staged
	require.Contains(t, html, `data-count="3"`)
	require.Less(t, strings.Index(html, fixture.hex(1)), strings.Index(html, c.Hex()))
	require.Empty(t, fixture.staged(t, fixture.userID))
}

// TestStepEditAttachments_AdoptionSkipsWhatIsNotTheCallersStagedFile leaves another user's staged file, the caller's own file in another category, and a saved file elsewhere where they are
func TestStepEditAttachments_AdoptionSkipsWhatIsNotTheCallersStagedFile(t *testing.T) {

	fixture := newEditFixture(t)
	otherUser := primitive.NewObjectID()
	otherObject := primitive.NewObjectID()
	theirs := fixture.stage(t, otherUser, "theirs.png", "image/png")

	icon := model.NewAttachment("User", fixture.userID)
	icon.Category = "icon"
	icon.Original = "icon.png"
	icon.ContentType = "image/png"
	require.NoError(t, fixture.session.Collection("Attachment").Save(&icon, "seed"))

	elsewhere := model.NewAttachment("Stream", otherObject)
	elsewhere.Category = "photo"
	elsewhere.Original = "elsewhere.png"
	elsewhere.ContentType = "image/png"
	require.NoError(t, fixture.session.Collection("Attachment").Save(&elsewhere, "seed"))

	order := []string{theirs.Hex(), icon.AttachmentID.Hex(), elsewhere.AttachmentID.Hex(), "garbage"}
	result, _ := fixture.post(photoEdit(0, 64), fixture.multipartRequest(t, "", order))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, fixture.photos(t))
	require.Len(t, fixture.staged(t, otherUser), 1)

	icons, err := fixture.service.CountByCategory(fixture.session, "User", fixture.userID, "icon")
	require.NoError(t, err)
	require.Equal(t, 1, icons)

	others, err := fixture.service.CountByCategory(fixture.session, "Stream", otherObject, "photo")
	require.NoError(t, err)
	require.Equal(t, 1, others)
}

// TestStepEditAttachments_AdoptionNeedsASignedInUser adopts nothing for a request with no user, even when the order names a staged file
func TestStepEditAttachments_AdoptionNeedsASignedInUser(t *testing.T) {

	fixture := newEditFixture(t)
	owner := fixture.userID
	staged := fixture.stage(t, owner, "a.png", "image/png")
	fixture.userID = primitive.NilObjectID

	result, _ := fixture.post(photoEdit(0, 64), fixture.multipartRequest(t, "", []string{staged.Hex()}))

	require.Nil(t, result.Error)
	require.Empty(t, fixture.photos(t))
	require.Len(t, fixture.staged(t, owner), 1)
}

// TestStepEditAttachments_AdoptionCountsAgainstTheMaximum refuses the post whole, and leaves the files staged, when the staged files would pass the maximum
func TestStepEditAttachments_AdoptionCountsAgainstTheMaximum(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)
	a := fixture.stage(t, fixture.userID, "a.png", "image/png")
	b := fixture.stage(t, fixture.userID, "b.png", "image/png")

	result, html := fixture.post(photoEdit(1, 3), fixture.multipartRequest(t, "", []string{a.Hex(), b.Hex()}))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.True(t, derp.IsValidationError(result.Error))
	require.Equal(t, "At most 3 files are allowed", derp.Message(result.Error))
	require.Empty(t, html)
	require.Len(t, fixture.photos(t), 2)
	require.Len(t, fixture.staged(t, fixture.userID), 2)
}

// TestStepEditAttachments_AdoptionRefusesAStagedFileOfTheWrongType halts with a 400 and adopts nothing when a staged file is not an accepted type
func TestStepEditAttachments_AdoptionRefusesAStagedFileOfTheWrongType(t *testing.T) {

	fixture := newEditFixture(t)
	fixture.object.CreateDate = 0
	a := fixture.stage(t, fixture.userID, "a.png", "image/png")
	notes := fixture.stage(t, fixture.userID, "notes.txt", "text/plain")

	result, _ := fixture.post(photoEdit(0, 64), fixture.multipartRequest(t, "", []string{a.Hex(), notes.Hex()}))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
	require.Empty(t, fixture.photos(t))
	require.Len(t, fixture.staged(t, fixture.userID), 2)
}
