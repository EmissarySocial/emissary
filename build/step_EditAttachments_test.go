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
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// stubEditBuilder adds what the edit-attachments step reads beyond the upload fixture's builder:
// the query string, the object's saved state, the action's own path and steps, and the signed-in user
type stubEditBuilder struct {
	stubAttachmentBuilder
	_object data.Object
	_userID primitive.ObjectID
	_action model.Action
}

// action returns the action being run, whose steps say whether the step was reached through include. Implements the Builder interface.
func (builder stubEditBuilder) action() model.Action {
	return builder._action
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
// unless a test clears its CreateDate, the user who is signed in, and the action being run, which
// reaches the step through include unless a test says the widget addressed the step's own action
type editFixture struct {
	uploadFixture
	object *model.Stream
	userID primitive.ObjectID
	action model.Action
}

// newEditFixture stores one "photo" Attachment for each rank given, on a Stream that counts as saved
func newEditFixture(t *testing.T, ranks ...int) editFixture {
	t.Helper()

	fixture := editFixture{uploadFixture: newUploadFixture(t, ranks...), userID: primitive.NewObjectID(), action: includingAction()}

	stream := model.NewStream()
	stream.StreamID = fixture.objectID
	stream.CreateDate = 1
	fixture.object = &stream

	return fixture
}

// newStagingFixture is the fixture as the widget's own request on a new object sees it: the
// Stream is not saved, and the action being run is the photos action that holds the step
func newStagingFixture(t *testing.T, ranks ...int) editFixture {
	t.Helper()

	fixture := newEditFixture(t, ranks...)
	fixture.object.CreateDate = 0
	fixture.action = widgetAction()

	return fixture
}

// includingAction is the create action as a form's Save reaches the step: through include, so
// the step is not among the action's own steps
func includingAction() model.Action {
	return model.Action{Steps: sliceof.Object[step.Step]{step.Include{Action: "photos"}}}
}

// widgetAction is the photos action as the widget's own requests reach the step: holding it,
// here inside a modal, as a create form's action would, with the save that finishes a saved object's change after it
func widgetAction() model.Action {
	return model.Action{Steps: sliceof.Object[step.Step]{step.AsModal{SubSteps: []step.Step{step.EditAttachments{Category: "photo", Fieldname: "photos"}}}, step.Save{}}}
}

// sentinelStep is a step with no build-side twin, so the pipeline halts with an error on
// reaching it: placed after edit-attachments, it marks a request that ran on when it should have stopped
type sentinelStep struct{}

// Name returns the step's name. Implements the step.Step interface.
func (sentinelStep) Name() string {
	return "sentinel"
}

// RequiredModel returns no model requirement. Implements the step.Step interface.
func (sentinelStep) RequiredModel() string {
	return ""
}

// RequiredStates returns no state requirement. Implements the step.Step interface.
func (sentinelStep) RequiredStates() []string {
	return nil
}

// RequiredRoles returns no role requirement. Implements the step.Step interface.
func (sentinelStep) RequiredRoles() []string {
	return nil
}

// stagingQuery is the query a create form's widget reaches the step with: the parent's add,
// naming the template and the step's own action
const stagingQuery = "?templateId=photo&action=photos"

// stagingRoute is the route the staging row and its tiles post back to, as the step re-encodes
// the request's own path and query and the builder escapes it
const stagingRoute = "/parent/add?action=photos&amp;templateId=photo"

// zoneOpen opens the zone the step draws on GET, which the behavior and the styles find
const zoneOpen = `<div class="upload-multiple" data-upload-multiple="true">`

// zoneControls is what follows the row inside a zone, as writeControls draws it for the photos field
const zoneControls = `<button id="upload-multiple-photos-add" type="button" class="upload-multiple-add" title="Add a file" aria-label="Add a file"><i class="bi bi-plus-lg"></i></button><input type="file" name="photos" id="upload-multiple-photos-input" accept="image/*" hidden="hidden" aria-label="Add a file"><div id="upload-multiple-photos-status" class="upload-multiple-status" role="status"></div>`

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
		_action: fixture.action,
	}
}

// path is the action's own URL
func (fixture editFixture) path() string {
	return "/" + fixture.objectID.Hex() + "/photos"
}

// get runs the step's GET and returns the result with what it wrote
func (fixture editFixture) get(step StepEditAttachments) (PipelineResult, string) {
	return fixture.getWithQuery(step, "")
}

// getWithQuery runs the step's GET with a query string and returns the result with what it wrote
func (fixture editFixture) getWithQuery(step StepEditAttachments, query string) (PipelineResult, string) {
	return fixture.getAt(step, fixture.path()+query)
}

// getAt runs the step's GET against the given URL and returns the result with what it wrote
func (fixture editFixture) getAt(step StepEditAttachments, target string) (PipelineResult, string) {

	request := httptest.NewRequest(http.MethodGet, target, nil)

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

// stagingRequest builds a multipart POST to the parent's add, with the query given, carrying each
// file under the given field beside a form field the step must ignore
func (fixture editFixture) stagingRequest(t *testing.T, query string, field string, files ...postedFile) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, file := range files {
		part, err := writer.CreateFormFile(field, file.name)
		require.NoError(t, err)
		_, err = part.Write([]byte(file.content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.WriteField("label", "ignored"))
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/parent/add"+query, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request
}

// formRequest builds a URL-encoded POST to the action's own URL
func (fixture editFixture) formRequest(query string, values url.Values) *http.Request {
	return formRequestTo(fixture.path()+query, values)
}

// formRequestTo builds a URL-encoded POST to the given URL
func formRequestTo(target string, values url.Values) *http.Request {

	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
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

// TestStepEditAttachments_GetDrawsTheZone writes the zone's contents: the tile row the widget
// expects, in rank order, with the limits on it, then the plus, the input, and the status line
func TestStepEditAttachments_GetDrawsTheZone(t *testing.T) {

	fixture := newEditFixture(t, 2)
	image := fixture.storeImage(t, 1)
	plain := fixture.byRank[2]

	result, html := fixture.get(photoEdit(1, 64))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	base := "/" + fixture.objectID.Hex()

	require.True(t, strings.HasPrefix(html, zoneOpen+`<div class="upload-multiple-tiles" data-url="`+base+`/photos" data-name="photos" data-accept="image/*" data-minimum="1" data-maximum="64" hx-params="photos,photos.order" hx-push-url="false" hx-sync="closest .upload-multiple:queue all" hx-encoding="multipart/form-data">`), html)
	require.True(t, strings.HasSuffix(html, "</div></div>"+zoneControls+"</div>"), html)

	// Every tile is its own handle and takes focus, with an X that shows on hover; an image tile
	// previews the file, and any other tile shows an icon and the filename
	require.Contains(t, html, `<div class="upload-multiple-tile hover-trigger" data-value="`+image.Hex()+`" data-group="image" tabindex="0" aria-label="Move beach.jpg"><img src="`+base+`/attachments/`+image.Hex()+`?width=300" class="upload-multiple-preview" alt="beach.jpg" draggable="false"><button type="button" class="upload-multiple-remove hover-show" hx-post="`+base+`/photos?attachmentId=`+image.Hex()+`" hx-params="none" hx-confirm="Delete this file?" hx-target="closest .upload-multiple-tiles" hx-swap="outerHTML" hx-push-url="false" hx-sync="closest .upload-multiple:queue all" aria-label="Delete beach.jpg"><i class="bi bi-x-lg"></i></button><input type="hidden" name="photos.order" value="`+image.Hex()+`"></div>`)
	require.Contains(t, html, `<div class="upload-multiple-tile hover-trigger" data-value="`+plain.Hex()+`" tabindex="0" aria-label="Move file.jpg"><i class="bi bi-file-earmark upload-multiple-icon"></i><span class="upload-multiple-filename">file.jpg</span>`)
	require.Less(t, strings.Index(html, image.Hex()), strings.Index(html, plain.Hex()))
	require.Equal(t, 2, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
}

// TestStepEditAttachments_GetReadOnlyDrawsTheFilesAlone writes the tiles with no handle, X, order
// input, or controls, on a row marked for the behavior to leave alone
func TestStepEditAttachments_GetReadOnlyDrawsTheFilesAlone(t *testing.T) {

	fixture := newEditFixture(t, 2)
	image := fixture.storeImage(t, 1)
	plain := fixture.byRank[2]

	result, html := fixture.getWithQuery(photoEdit(1, 64), "?readonly=true")

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	base := "/" + fixture.objectID.Hex()

	require.Equal(t, zoneOpen+`<div class="upload-multiple-tiles" data-readonly="true"><div class="upload-multiple-tile" data-value="`+image.Hex()+`" data-group="image"><img src="`+base+`/attachments/`+image.Hex()+`?width=300" class="upload-multiple-preview" alt="beach.jpg" draggable="false"></div><div class="upload-multiple-tile" data-value="`+plain.Hex()+`"><i class="bi bi-file-earmark upload-multiple-icon"></i><span class="upload-multiple-filename">file.jpg</span></div></div></div>`, html)
}

// TestStepEditAttachments_PostAnswersWithTheRowAlone writes the row and nothing after it, so the
// plus and the input the GET drew stay in place while the row swaps
func TestStepEditAttachments_PostAnswersWithTheRowAlone(t *testing.T) {

	fixture := newEditFixture(t, 1, 2)

	result, html := fixture.post(photoEdit(1, 64), fixture.formRequest("", url.Values{"photos.order": {fixture.hex(2), fixture.hex(1)}}))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.True(t, strings.HasPrefix(html, `<div class="upload-multiple-tiles" data-url=`), html)
	require.True(t, strings.HasSuffix(html, "</div></div>"), html)
	require.NotContains(t, html, `class="upload-multiple-add"`)
	require.NotContains(t, html, `type="file"`)
	require.Equal(t, 1, fixture.rankOf(t, 2))
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

// TestStepEditAttachments_GetOnANewObjectDrawsTheStagingZone draws, for the widget's own request
// on a new object, an empty row wired to stage each file back through the same route, then the
// controls, and never carries a stale attachmentId into the row
func TestStepEditAttachments_GetOnANewObjectDrawsTheStagingZone(t *testing.T) {

	fixture := newStagingFixture(t)
	row := `<div class="upload-multiple-tiles" data-staging-url="` + stagingRoute + `" data-name="photos" data-accept="image/*" data-minimum="1" data-maximum="16" hx-params="photos" hx-push-url="false" hx-sync="closest .upload-multiple:queue all" hx-encoding="multipart/form-data"></div>`

	result, html := fixture.getAt(photoEdit(1, 16), "/parent/add"+stagingQuery)

	require.Nil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, zoneOpen+row+zoneControls+"</div>", html)

	result, html = fixture.getAt(photoEdit(1, 16), "/parent/add"+stagingQuery+"&attachmentId=stale")

	require.Nil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, zoneOpen+row+zoneControls+"</div>", html)
}

// TestStepEditAttachments_StagesTheFileUnderTheUserAndDrawsItsTile files the one file the
// widget posts on a new object under the signed-in user, in the staging category, and answers
// with its tile alone: previewed from the user, with an X that posts back here and replaces the
// tile, and an order input the form's Save will post
func TestStepEditAttachments_StagesTheFileUnderTheUserAndDrawsItsTile(t *testing.T) {

	fixture := newStagingFixture(t)

	result, html := fixture.post(photoEdit(1, 16), fixture.stagingRequest(t, stagingQuery, "photos", png("a.png")))

	require.Nil(t, result.Error)
	require.True(t, result.Halt)

	staged := fixture.staged(t, fixture.userID)
	require.Len(t, staged, 1)
	require.Equal(t, "User", staged[0].ObjectType)
	require.Equal(t, fixture.userID, staged[0].ObjectID)
	require.Equal(t, "upload-staging", staged[0].Category)
	require.Equal(t, "a.png", staged[0].Original)
	require.Equal(t, "image/png", staged[0].ContentType)
	require.Empty(t, fixture.photos(t))
	fixture.requireStored(t, staged...)

	id := staged[0].AttachmentID.Hex()
	require.True(t, strings.HasPrefix(html, `<div class="upload-multiple-tile hover-trigger" data-value="`+id+`" data-group="image" tabindex="0" aria-label="Move a.png"><img src="/@`+fixture.userID.Hex()+`/attachments/`+id+`?width=300" class="upload-multiple-preview" alt="a.png" draggable="false"><button type="button" class="upload-multiple-remove hover-show" hx-post="`+stagingRoute+`&amp;attachmentId=`+id+`" hx-params="none"`), html)
	require.Contains(t, html, `hx-target="closest .upload-multiple-tile" hx-swap="outerHTML"`)
	require.NotContains(t, html, "Delete this file?")
	require.True(t, strings.HasSuffix(html, `<input type="hidden" name="photos.order" value="`+id+`"></div>`), html)
	require.Equal(t, 1, strings.Count(html, "upload-multiple-tile hover-trigger"))
	require.NotContains(t, html, "upload-multiple-tiles")
}

// TestStepEditAttachments_StagingStopsTheAction runs the step through the pipeline with a step
// after it that fails when reached. The widget's requests on a new object stop at the step,
// because what follows belongs to the saved object and a save there would write the empty object
// once per file; the form's Save through include runs on to the steps that finish it
func TestStepEditAttachments_StagingStopsTheAction(t *testing.T) {

	pipeline := Pipeline{step.EditAttachments(photoEdit(1, 16)), sentinelStep{}}
	fixture := newStagingFixture(t)
	var buffer bytes.Buffer

	// The widget's GET answers with the zone and stops
	request := httptest.NewRequest(http.MethodGet, "/parent/add"+stagingQuery, nil)
	result := pipeline.Get(fixture.builder(request).factory(), fixture.builder(request), &buffer)

	require.Nil(t, result.Error)
	require.True(t, result.Halt)
	require.Contains(t, buffer.String(), `data-staging-url="`+stagingRoute+`"`)

	// The widget's upload answers with the tile and stops
	buffer.Reset()
	request = fixture.stagingRequest(t, stagingQuery, "photos", png("a.png"))
	result = pipeline.Post(fixture.builder(request).factory(), fixture.builder(request), &buffer)

	require.Nil(t, result.Error)
	require.True(t, result.Halt)
	require.Contains(t, buffer.String(), `class="upload-multiple-tile hover-trigger"`)
	require.Len(t, fixture.staged(t, fixture.userID), 1)

	// The form's Save on the new object adopts what it names and runs on
	saving := newEditFixture(t)
	saving.object.CreateDate = 0
	a := saving.stage(t, saving.userID, "a.png", "image/png")

	buffer.Reset()
	request = saving.multipartRequest(t, "", []string{a.Hex()})
	result = pipeline.Post(saving.builder(request).factory(), saving.builder(request), &buffer)

	require.ErrorContains(t, result.Error, "Unrecognized Pipeline Step")
	require.Empty(t, buffer.String())
	require.Len(t, saving.photos(t), 1)
}

// TestStepEditAttachments_StagingRefusesAWrongType refuses a file the step does not accept at
// once, so the form hears it now and not at its Save, and stores nothing
func TestStepEditAttachments_StagingRefusesAWrongType(t *testing.T) {

	fixture := newStagingFixture(t)

	result, html := fixture.post(photoEdit(1, 16), fixture.stagingRequest(t, stagingQuery, "photos", text("notes.txt")))

	require.Error(t, result.Error)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
	require.Empty(t, html)
	require.Empty(t, fixture.staged(t, fixture.userID))
	require.Zero(t, fixture.storedFiles(t))
}

// TestStepEditAttachments_StagingRefusesNoFileOrMoreThanOne answers 400 and stores nothing for a
// body with no file under the step's field, or with more than one
func TestStepEditAttachments_StagingRefusesNoFileOrMoreThanOne(t *testing.T) {

	fixture := newStagingFixture(t)

	check := func(name string, request *http.Request) {
		t.Helper()

		result, html := fixture.post(photoEdit(1, 16), request)

		require.Error(t, result.Error, name)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), name)
		require.Empty(t, html, name)
		require.Empty(t, fixture.staged(t, fixture.userID), name)
		require.Zero(t, fixture.storedFiles(t), name)
	}

	check("no file", fixture.stagingRequest(t, stagingQuery, "photos"))
	check("two files", fixture.stagingRequest(t, stagingQuery, "photos", png("a.png"), png("b.png")))
	check("a file under another field", fixture.stagingRequest(t, stagingQuery, "other", png("a.png")))
}

// TestStepEditAttachments_StagingRefusesASignedOutRequest answers 401 and stores nothing when no user is signed in
func TestStepEditAttachments_StagingRefusesASignedOutRequest(t *testing.T) {

	fixture := newStagingFixture(t)
	fixture.userID = primitive.NilObjectID

	result, html := fixture.post(photoEdit(1, 16), fixture.stagingRequest(t, stagingQuery, "photos", png("a.png")))

	require.Error(t, result.Error)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(result.Error))
	require.Empty(t, html)
	require.Zero(t, fixture.storedFiles(t))
}

// TestStepEditAttachments_StagedDeleteRefusals answers 400 and deletes nothing for an ID that is
// not one of the user's own staged files: an invalid token, another user's staged file, or a
// file that is already on an object. The delete that goes through needs the media server the
// fixture's Attachment service has no way to carry, so it is driven in the browser instead.
func TestStepEditAttachments_StagedDeleteRefusals(t *testing.T) {

	fixture := newStagingFixture(t, 1)
	other := primitive.NewObjectID()
	theirs := fixture.stage(t, other, "b.png", "image/png")
	mine := fixture.stage(t, fixture.userID, "a.png", "image/png")
	photo := fixture.byRank[1]

	check := func(name string, token string) {
		t.Helper()

		result, html := fixture.post(photoEdit(1, 16), formRequestTo("/parent/add"+stagingQuery+"&attachmentId="+token, url.Values{}))

		require.Error(t, result.Error, name)
		require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error), name)
		require.Empty(t, html, name)
		require.Equal(t, []primitive.ObjectID{mine}, idsOf(fixture.staged(t, fixture.userID)), name)
		require.Equal(t, []primitive.ObjectID{theirs}, idsOf(fixture.staged(t, other)), name)
		require.Len(t, fixture.photos(t), 1, name)
	}

	check("an invalid token", "not-an-id")
	check("another user's staged file", theirs.Hex())
	check("a file on the object", photo.Hex())
}

// idsOf returns the IDs of the given attachments, in order
func idsOf(attachments []model.Attachment) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0, len(attachments))

	for _, attachment := range attachments {
		result = append(result, attachment.AttachmentID)
	}

	return result
}

// TestReachedThroughInclude tells a request that names the step's own action, at any depth,
// from one that reached the step through another action's include
func TestReachedThroughInclude(t *testing.T) {

	own := step.EditAttachments{Category: "photo", Fieldname: "photos"}

	check := func(name string, steps []step.Step, want bool) {
		t.Helper()
		builder := stubEditBuilder{_action: model.Action{Steps: steps}}
		require.Equal(t, want, reachedThroughInclude(builder, "photos"), name)
	}

	check("the step itself", []step.Step{own}, false)
	check("inside a modal", []step.Step{step.AsModal{SubSteps: []step.Step{own}}}, false)
	check("inside a condition's other branch", []step.Step{step.IfCondition{Otherwise: []step.Step{own}}}, false)
	check("inside a switch case", []step.Step{step.Switch{Cases: map[string][]step.Step{"gallery": {own}}}}, false)
	check("through include", []step.Step{step.Include{Action: "photos"}}, true)
	check("another field's step", []step.Step{step.EditAttachments{Category: "cover", Fieldname: "covers"}}, true)
	check("no steps", nil, true)
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
	require.Equal(t, 1, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
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
	require.Contains(t, html, `data-maximum="2"`)
	require.Equal(t, 3, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
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

	// keeps names the files kept in rank order, the posted ones and then the staged ones
	keeps := func(name string, minimum int, maximum int, existing int, files []*multipart.FileHeader, stagedFiles []model.Attachment, want ...string) {
		t.Helper()

		gotFiles, gotStaged, err := photoEdit(minimum, maximum).limitSave(existing, files, stagedFiles)
		require.NoError(t, err, name)

		kept := make([]string, 0, len(gotFiles)+len(gotStaged))
		for _, file := range gotFiles {
			kept = append(kept, file.Filename)
		}

		for _, attachment := range gotStaged {
			kept = append(kept, attachment.Original)
		}

		require.Equal(t, append([]string{}, want...), kept, name)
	}

	refuses := func(name string, minimum int, maximum int, existing int, files []*multipart.FileHeader, stagedFiles []model.Attachment) {
		t.Helper()

		gotFiles, gotStaged, err := photoEdit(minimum, maximum).limitSave(existing, files, stagedFiles)
		require.True(t, derp.IsValidationError(err), name)
		require.Equal(t, minimumMessage(minimum), derp.Message(err), name)
		require.Nil(t, gotFiles, name)
		require.Nil(t, gotStaged, name)
	}

	keeps("everything fits", 1, 4, 0, posted("a", "b"), staged("c", "d"), "a", "b", "c", "d")
	keeps("the staged tail is dropped", 1, 3, 0, posted("a", "b"), staged("c", "d"), "a", "b", "c")
	keeps("every staged file is dropped", 1, 2, 0, posted("a", "b"), staged("c", "d"), "a", "b")
	keeps("the posted tail is dropped", 1, 1, 0, posted("a", "b"), staged("c"), "a")
	keeps("staged alone", 1, 2, 0, nil, staged("a", "b", "c"), "a", "b")
	keeps("existing files take room first", 1, 3, 2, posted("a"), staged("b"), "a")
	keeps("existing files over the maximum leave no room", 1, 2, 3, posted("a"), staged("b"))
	keeps("nothing posted with no minimum", 0, 2, 0, nil, nil)
	keeps("exactly the minimum", 2, 2, 0, posted("a"), staged("b"), "a", "b")

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
	require.Equal(t, 3, strings.Count(html, `class="upload-multiple-tile hover-trigger"`))
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

// TestAttachmentIDs keeps the values that are ObjectIDs, in order, and drops the rest
func TestAttachmentIDs(t *testing.T) {

	first := primitive.NewObjectID()
	second := primitive.NewObjectID()

	check := func(name string, values []string, expected []primitive.ObjectID) {
		t.Helper()
		require.Equal(t, expected, attachmentIDs(values), name)
	}

	check("nil", nil, []primitive.ObjectID{})
	check("empty", []string{}, []primitive.ObjectID{})
	check("ids in order", []string{second.Hex(), first.Hex()}, []primitive.ObjectID{second, first})
	check("values that are not IDs dropped", []string{"new:0", first.Hex(), "photo"}, []primitive.ObjectID{first})
	check("garbage dropped", []string{"", "garbage", "0123456789abcdef0123456", first.Hex()}, []primitive.ObjectID{first})
	check("uppercase hex", []string{strings.ToUpper(first.Hex())}, []primitive.ObjectID{first})
	check("repeats kept", []string{first.Hex(), first.Hex()}, []primitive.ObjectID{first, first})
}
