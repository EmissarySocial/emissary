package build

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/mediaserver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// stubUploadFactory satisfies the Factory interface by embedding it (nil) and overriding the
// services the attachment steps use
type stubUploadFactory struct {
	Factory
	attachment *service.Attachment
	media      mediaserver.MediaServer
	template   *service.Template
}

// Attachment returns the Attachment service under test. Implements the Factory interface.
func (factory stubUploadFactory) Attachment() *service.Attachment {
	return factory.attachment
}

// Template returns the Template service holding the test's templates, which only the staging
// step's GET reads. Implements the Factory interface.
func (factory stubUploadFactory) Template() *service.Template {
	return factory.template
}

// MediaServer returns the in-memory media server. Implements the Factory interface.
func (factory stubUploadFactory) MediaServer() mediaserver.MediaServer {
	return factory.media
}

// stubUploadBuilder adds the object the upload step reads, which these tests never write into
type stubUploadBuilder struct {
	stubAttachmentBuilder
}

// object returns an empty Stream, which no test writes into because none sets an attachment path. Implements the Builder interface.
func (builder stubUploadBuilder) object() data.Object {
	stream := model.NewStream()
	return &stream
}

// uploadFixture is the attachment fixture plus a media server over an in-memory filesystem
type uploadFixture struct {
	attachmentFixture
	originals afero.Fs
	media     mediaserver.MediaServer
}

// newUploadFixture stores one "photo" Attachment for each rank given, with nothing in the media server
func newUploadFixture(t *testing.T, ranks ...int) uploadFixture {
	t.Helper()

	originals := afero.NewMemMapFs()

	return uploadFixture{
		attachmentFixture: newAttachmentFixture(t, ranks...),
		originals:         originals,
		media:             mediaserver.New(originals, afero.NewMemMapFs(), mediaserver.NewWorkingDirectory(t.TempDir(), time.Minute, 10)),
	}
}

// postedFile is one file to include in a multipart upload
type postedFile struct {
	name    string
	content string
}

// png returns a file that http.DetectContentType recognizes as a PNG
func png(name string) postedFile {
	return postedFile{name: name, content: "\x89PNG\r\n\x1a\n" + strings.Repeat("payload-bytes", 100)}
}

// text returns a file that is plain text, whatever its name says
func text(name string) postedFile {
	return postedFile{name: name, content: strings.Repeat("this is plain text, not an image. ", 20)}
}

// post runs the step against a multipart request carrying the files under data.photos
func (fixture uploadFixture) post(t *testing.T, step StepUploadAttachments, files ...postedFile) PipelineResult {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, file := range files {
		part, err := writer.CreateFormFile("data.photos", file.name)
		require.NoError(t, err)
		_, err = part.Write([]byte(file.content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/000000000000000000000000/edit", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	builder := stubUploadBuilder{stubAttachmentBuilder{
		_request:  request,
		_session:  fixture.session,
		_factory:  stubUploadFactory{attachment: fixture.service, media: fixture.media},
		_objectID: fixture.objectID,
	}}

	var buffer bytes.Buffer
	return applyBehavior(step.Post(builder, &buffer))
}

// photos returns the category's Attachments in rank order
func (fixture uploadFixture) photos(t *testing.T) []model.Attachment {
	t.Helper()

	result, err := fixture.service.QueryByCategory(fixture.session, "Stream", fixture.objectID, "photo")
	require.NoError(t, err)

	return result
}

// storedFiles counts the files the media server holds
func (fixture uploadFixture) storedFiles(t *testing.T) int {
	t.Helper()

	count := 0

	require.NoError(t, afero.Walk(fixture.originals, "/", func(_ string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if !info.IsDir() {
			count++
		}

		return nil
	}))

	return count
}

// fillStep returns an upload step that keeps every existing photo and accepts images only
func fillStep(minimum int, maximum int) StepUploadAttachments {
	return StepUploadAttachments{Action: "fill", Fieldname: "data.photos", AcceptType: "image/*", Category: "photo", Maximum: maximum, Minimum: minimum}
}

// TestStepUploadAttachments_RanksAfterExisting gives new files the ranks after the category's highest, in posted order
func TestStepUploadAttachments_RanksAfterExisting(t *testing.T) {

	fixture := newUploadFixture(t, 1, 2)
	result := fixture.post(t, fillStep(0, 10), png("a.png"), png("b.png"))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)

	photos := fixture.photos(t)
	require.Len(t, photos, 4)
	require.Equal(t, []int{1, 2, 3, 4}, []int{photos[0].Rank, photos[1].Rank, photos[2].Rank, photos[3].Rank})
	require.Equal(t, "a.png", photos[2].Original)
	require.Equal(t, "b.png", photos[3].Original)
	require.Equal(t, "image/png", photos[2].ContentType)

	for _, photo := range photos[2:] {
		exists, err := afero.Exists(fixture.originals, photo.AttachmentID.Hex())
		require.NoError(t, err)
		require.True(t, exists, photo.Original)
	}
}

// TestStepUploadAttachments_RanksPastAGap counts from the highest rank, not from the number of files
func TestStepUploadAttachments_RanksPastAGap(t *testing.T) {

	fixture := newUploadFixture(t, 10, 40)
	result := fixture.post(t, fillStep(0, 10), png("a.png"))

	require.Nil(t, result.Error)

	photos := fixture.photos(t)
	require.Len(t, photos, 3)
	require.Equal(t, 41, photos[2].Rank)
}

// TestStepUploadAttachments_ReplaceStartsAtOne numbers the files of a replacing upload from 1
func TestStepUploadAttachments_ReplaceStartsAtOne(t *testing.T) {

	fixture := newUploadFixture(t)
	step := fillStep(0, 10)
	step.Action = "replace"
	result := fixture.post(t, step, png("a.png"), png("b.png"))

	require.Nil(t, result.Error)

	photos := fixture.photos(t)
	require.Len(t, photos, 2)
	require.Equal(t, 1, photos[0].Rank)
	require.Equal(t, "a.png", photos[0].Original)
	require.Equal(t, 2, photos[1].Rank)
	require.Equal(t, "b.png", photos[1].Original)
}

// TestStepUploadAttachments_UnrankedFilesStayFirst ranks a new file after files stored before ranks existed
func TestStepUploadAttachments_UnrankedFilesStayFirst(t *testing.T) {

	fixture := newUploadFixture(t)
	older := fixture.store(t, "photo", 0, "")
	newer := fixture.store(t, "photo", 0, "")

	result := fixture.post(t, fillStep(0, 10), png("c.png"))
	require.Nil(t, result.Error)

	photos := fixture.photos(t)
	require.Len(t, photos, 3)
	require.Equal(t, older, photos[0].AttachmentID)
	require.Equal(t, newer, photos[1].AttachmentID)
	require.Equal(t, "c.png", photos[2].Original)
	require.Equal(t, 1, photos[2].Rank)
}

// TestStepUploadAttachments_ChecksEveryFileBeforeStoringAny stores nothing when a later file is refused
func TestStepUploadAttachments_ChecksEveryFileBeforeStoringAny(t *testing.T) {

	fixture := newUploadFixture(t, 1)
	result := fixture.post(t, fillStep(0, 10), png("a.png"), text("notes.jpg"))

	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.Equal(t, http.StatusBadRequest, derp.ErrorCode(result.Error))
	require.Len(t, fixture.photos(t), 1)
	require.Zero(t, fixture.storedFiles(t))
}

// TestStepUploadAttachments_Minimum refuses an upload that would leave the category below the minimum
func TestStepUploadAttachments_Minimum(t *testing.T) {

	// No file on a create form with nothing stored yet
	empty := newUploadFixture(t)
	result := empty.post(t, fillStep(1, 10))
	require.NotNil(t, result.Error)
	require.True(t, result.Halt)
	require.True(t, derp.IsValidationError(result.Error))
	require.Equal(t, "At least 1 file is required", derp.RootMessage(result.Error))

	// No file on an edit form whose post already holds one
	holding := newUploadFixture(t, 1)
	result = holding.post(t, fillStep(1, 10))
	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Len(t, holding.photos(t), 1)

	// One file where three are required, with one stored: nothing is written
	short := newUploadFixture(t, 1)
	result = short.post(t, fillStep(3, 10), png("a.png"))
	require.True(t, derp.IsValidationError(result.Error))
	require.Equal(t, "At least 3 files are required", derp.RootMessage(result.Error))
	require.Len(t, short.photos(t), 1)
	require.Zero(t, short.storedFiles(t))

	// A replacement that would leave fewer than the minimum deletes nothing
	replacing := newUploadFixture(t, 1, 2, 3)
	step := fillStep(2, 10)
	step.Action = "replace"
	result = replacing.post(t, step, png("a.png"))
	require.True(t, derp.IsValidationError(result.Error))
	require.Len(t, replacing.photos(t), 3)
	require.Zero(t, replacing.storedFiles(t))
}

// TestStepUploadAttachments_NoFilesContinues does nothing when the form carries no file and no minimum applies
func TestStepUploadAttachments_NoFilesContinues(t *testing.T) {

	fixture := newUploadFixture(t, 1)
	result := fixture.post(t, fillStep(0, 10))

	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Len(t, fixture.photos(t), 1)
	require.Zero(t, fixture.storedFiles(t))
}

// TestCountAfterUpload pins how many files each action leaves in the category
func TestCountAfterUpload(t *testing.T) {

	check := func(action string, maximum int, existing int, posted int, expected int) {
		t.Helper()
		require.Equal(t, expected, countAfterUpload(action, maximum, existing, posted), "%s max %d existing %d posted %d", action, maximum, existing, posted)
	}

	check("replace", 10, 5, 2, 2)
	check("replace", 10, 0, 0, 0)
	check("fill", 10, 5, 2, 7)
	check("fill", 10, 0, 0, 0)
	check("append", 10, 5, 2, 7)
	check("append", 1, 1, 1, 1)
	check("append", 3, 2, 2, 3)
	check("", 3, 2, 2, 3)
	check("anything-else", 1, 0, 0, 0)
}

// TestNextRankAfter pins where the first new file lands
func TestNextRankAfter(t *testing.T) {

	ranked := func(ranks ...int) []model.Attachment {
		result := make([]model.Attachment, len(ranks))
		for index, rank := range ranks {
			result[index].Rank = rank
		}
		return result
	}

	require.Equal(t, 1, nextRankAfter("replace", ranked(5, 9)))
	require.Equal(t, 1, nextRankAfter("fill", ranked()))
	require.Equal(t, 1, nextRankAfter("append", nil))
	require.Equal(t, 10, nextRankAfter("fill", ranked(5, 9)))
	require.Equal(t, 10, nextRankAfter("append", ranked(5, 9)))
	require.Equal(t, 1, nextRankAfter("fill", ranked(0, 0)))
	require.Equal(t, 1, nextRankAfter("fill", ranked(-3)))
}

// TestMinimumMessage pins the inline message, singular and plural
func TestMinimumMessage(t *testing.T) {
	require.Equal(t, "At least 1 file is required", minimumMessage(1))
	require.Equal(t, "At least 2 files are required", minimumMessage(2))
	require.Equal(t, "At least 64 files are required", minimumMessage(64))
}
