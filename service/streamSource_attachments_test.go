package service

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * In-Memory Fakes
 ******************************************/

// fakeAttachments stands in for the Attachment service, holding every record it was given,
// including the deleted ones
type fakeAttachments struct {
	records []model.Attachment
	saves   int
	deletes int
}

// QueryByCategory implements the attachmentWriter interface
func (store *fakeAttachments) QueryByCategory(_ data.Session, objectType string, objectID primitive.ObjectID, category string) ([]model.Attachment, error) {

	result := make([]model.Attachment, 0)

	for _, record := range store.records {
		if (record.ObjectType == objectType) && (record.ObjectID == objectID) && (record.Category == category) && !record.IsDeleted() {
			result = append(result, record)
		}
	}

	return result, nil
}

// LoadByID implements the attachmentWriter interface
func (store *fakeAttachments) LoadByID(_ data.Session, objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID, result *model.Attachment) error {

	for _, record := range store.records {
		if (record.AttachmentID == attachmentID) && (record.ObjectType == objectType) && (record.ObjectID == objectID) && !record.IsDeleted() {
			*result = record
			return nil
		}
	}

	return derp.NotFound("test", "Attachment not found", attachmentID)
}

// Save implements the attachmentWriter interface, calculating the URL as the real service does
func (store *fakeAttachments) Save(_ data.Session, attachment *model.Attachment, _ string) error {

	store.saves++
	attachment.URL = attachment.CalcURL("https://site.example")

	for index, record := range store.records {
		if record.AttachmentID == attachment.AttachmentID {
			store.records[index] = *attachment
			return nil
		}
	}

	store.records = append(store.records, *attachment)
	return nil
}

// Delete implements the attachmentWriter interface (virtual delete)
func (store *fakeAttachments) Delete(_ data.Session, attachment *model.Attachment, _ string) error {

	store.deletes++

	for index, record := range store.records {
		if record.AttachmentID == attachment.AttachmentID {
			store.records[index].DeleteDate = 1
		}
	}

	return nil
}

// live returns the records that are not deleted
func (store *fakeAttachments) live() []model.Attachment {

	result := make([]model.Attachment, 0)

	for _, record := range store.records {
		if !record.IsDeleted() {
			result = append(result, record)
		}
	}

	return result
}

// fakeMedia stands in for the MediaServer, holding every file written to it
type fakeMedia struct {
	files    map[string]string
	putError error
	deleted  []string
}

// Put implements the mediaWriter interface.  It reads the whole file, as the real one does, so
// an error from the reader surfaces here.
func (media *fakeMedia) Put(filename string, file io.Reader) error {

	if media.putError != nil {
		return media.putError
	}

	body, err := io.ReadAll(file)

	if err != nil {
		return err
	}

	if media.files == nil {
		media.files = make(map[string]string)
	}

	media.files[filename] = string(body)
	return nil
}

// Delete implements the mediaWriter interface
func (media *fakeMedia) Delete(filename string) error {
	media.deleted = append(media.deleted, filename)
	return nil
}

/******************************************
 * Test Helpers
 ******************************************/

// attachmentSourceURL is the raw file that the attachment tests synchronize
const attachmentSourceURL = "https://raw.example.com/owner/repo/main/docs/guide.md"

// attachmentURL returns the address that a link to attachments/<name> resolves to
func attachmentURL(name string) string {
	return "https://raw.example.com/owner/repo/main/docs/attachments/" + name
}

// Minimal files whose bytes identify their type
const (
	pngFile = "\x89PNG\r\n\x1a\n" + "\x00\x00\x00\x0dIHDR"
	mp4File = "\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom"
	pdfFile = "%PDF-1.4\n%test"
)

// newAttachmentSync returns a sync service whose source links to files, plus its fakes
func newAttachmentSync(t *testing.T, file string) (*StreamSource, streamSourceSession, model.StreamSource, *fakeAttachments, *fakeStreamWriter) {

	t.Helper()

	adapter := &fakeAdapter{item: markdownItem(t, file)}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)
	streamSource.URL = attachmentSourceURL

	store := &fakeAttachments{}
	service.attachmentService = store

	return service, session, streamSource, store, writer
}

// attachmentTasks returns the IDs of the attachment downloads that a session queued
func attachmentTasks(session streamSourceSession) []string {

	result := make([]string, 0)

	for _, task := range session.publishedTasksNamed(TaskSyncStreamSourceAttachment) {
		result = append(result, task.Arguments.GetString("attachmentId"))
	}

	return result
}

/******************************************
 * Stage One: Reconciling Inside the Sync
 ******************************************/

// TestStreamSourceAttachments_NewLinks confirms that every linked file gets a WORKING attachment,
// that the saved content points at the local copies, and that one download is queued for each
func TestStreamSourceAttachments_NewLinks(t *testing.T) {

	service, session, streamSource, store, writer := newAttachmentSync(t, "![Flow](attachments/flow.png)\n\n[Manual](attachments/manual.pdf)\n")

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	live := store.live()
	require.Len(t, live, 2)

	for rank, name := range []string{"flow.png", "manual.pdf"} {
		require.Equal(t, attachmentURL(name), live[rank].SourceURL)
		require.Equal(t, name, live[rank].Original)
		require.Equal(t, model.AttachmentCategoryStreamSource, live[rank].Category)
		require.Equal(t, model.AttachmentStatusWorking, live[rank].Status)
		require.Equal(t, rank, live[rank].Rank)
		require.Equal(t, streamSource.StreamID, live[rank].ObjectID)
	}

	require.Len(t, writer.saved, 1)
	html := writer.saved[0].Content.HTML
	require.Contains(t, html, `src="`+live[0].URL+`"`)
	require.Contains(t, html, `href="`+live[1].URL+`"`)
	require.NotContains(t, html, "attachments/flow.png")
	require.Equal(t, "![Flow](attachments/flow.png)\n\n[Manual](attachments/manual.pdf)\n", writer.saved[0].Content.Raw, "the source keeps its own links")

	tasks := session.publishedTasksNamed(TaskSyncStreamSourceAttachment)
	require.Len(t, tasks, 2)
	require.Equal(t, "StreamSourceAttachment:"+live[0].AttachmentID.Hex(), tasks[0].Signature)
	require.Equal(t, streamSource.StreamID.Hex(), tasks[0].Arguments.GetString("streamId"))
}

// TestStreamSourceAttachments_VideoBecomesAPlayer confirms that an image link to a video is saved
// as a <video> element pointing at the local copy
func TestStreamSourceAttachments_VideoBecomesAPlayer(t *testing.T) {

	service, session, streamSource, store, writer := newAttachmentSync(t, "![Demo](attachments/demo.mp4)\n")

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Contains(t, writer.saved[0].Content.HTML, `<video controls src="`+store.live()[0].URL+`">Demo</video>`)
}

// TestStreamSourceAttachments_ReusedAcrossSyncs confirms that a file already imported is reused
// rather than imported again.  Two servers syncing one record both write the Stream, so the
// database rejects one, and its retry must find the other's attachments and reuse them.
func TestStreamSourceAttachments_ReusedAcrossSyncs(t *testing.T) {

	service, session, streamSource, store, writer := newAttachmentSync(t, "![Flow](attachments/flow.png)\n")

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))
	first := store.live()[0]
	saves := store.saves

	// The page changes, and still links to the same file
	service.adapters[model.StreamSourceMethodHTTPS].(*fakeAdapter).item = markdownItem(t, "# New title\n\n![Flow](attachments/flow.png)\n")
	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, store.live(), 1, "one attachment per file, however many syncs")
	require.Equal(t, first.AttachmentID, store.live()[0].AttachmentID)
	require.Equal(t, saves, store.saves, "an unmoved attachment is not written again")
	require.Contains(t, writer.saved[1].Content.HTML, first.URL)
}

// TestStreamSourceAttachments_RemovedLinkIsDeleted confirms that a file the page no longer links
// to is deleted, and that one it still links to is kept and re-ranked
func TestStreamSourceAttachments_RemovedLinkIsDeleted(t *testing.T) {

	service, session, streamSource, store, _ := newAttachmentSync(t, "![A](attachments/a.png)\n\n![B](attachments/b.png)\n")

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))
	require.Len(t, store.live(), 2)

	service.adapters[model.StreamSourceMethodHTTPS].(*fakeAdapter).item = markdownItem(t, "![B](attachments/b.png)\n")
	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	live := store.live()
	require.Len(t, live, 1)
	require.Equal(t, attachmentURL("b.png"), live[0].SourceURL)
	require.Equal(t, 0, live[0].Rank, "b.png moved to the top")
}

// TestStreamSourceAttachments_DuplicatesAreDeleted confirms that a second attachment for one
// address, however it arose, is removed on the next sync
func TestStreamSourceAttachments_DuplicatesAreDeleted(t *testing.T) {

	service, session, streamSource, store, _ := newAttachmentSync(t, "![A](attachments/a.png)\n")

	for range 2 {
		duplicate := newImportedAttachment(streamSource.StreamID, attachmentURL("a.png"))
		require.NoError(t, store.Save(nil, &duplicate, "test"))
	}

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))
	require.Len(t, store.live(), 1)
}

// TestStreamSourceAttachments_OtherCategoriesUntouched confirms that reconciling never deletes an
// attachment the sync did not create
func TestStreamSourceAttachments_OtherCategoriesUntouched(t *testing.T) {

	service, session, streamSource, store, _ := newAttachmentSync(t, "# No links\n")

	uploaded := model.NewAttachment(model.AttachmentObjectTypeStream, streamSource.StreamID)
	uploaded.Category = "image"
	require.NoError(t, store.Save(nil, &uploaded, "test"))

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))
	require.Len(t, store.live(), 1)
	require.Zero(t, store.deletes)
}

// TestStreamSourceAttachments_Cap confirms that links past the cap stay remote
func TestStreamSourceAttachments_Cap(t *testing.T) {

	var file strings.Builder

	for index := range 51 {
		file.WriteString("![x](attachments/" + strconv.Itoa(index) + ".png)\n\n")
	}

	service, session, streamSource, store, writer := newAttachmentSync(t, file.String())

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, store.live(), 50)
	require.Contains(t, writer.saved[0].Content.HTML, `src="attachments/50.png"`, "the fifty-first link stays a link")
}

// TestStreamSourceAttachments_UnchangedSyncRetries confirms that a sync ending on a 304 still
// re-queues every file that is not READY, and none that is
func TestStreamSourceAttachments_UnchangedSyncRetries(t *testing.T) {

	service, session, streamSource, store, _ := newAttachmentSync(t, "unused")
	service.adapters[model.StreamSourceMethodHTTPS].(*fakeAdapter).version = `"same"`
	streamSource.Version = `"same"`

	statuses := []string{model.AttachmentStatusReady, model.AttachmentStatusWorking, model.AttachmentStatusFailed}
	ids := make([]string, 0, len(statuses))

	for _, status := range statuses {
		attachment := newImportedAttachment(streamSource.StreamID, attachmentURL(status+".png"))
		attachment.Status = status
		require.NoError(t, store.Save(nil, &attachment, "test"))
		ids = append(ids, attachment.AttachmentID.Hex())
	}

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Equal(t, ids[1:], attachmentTasks(session), "WORKING and FAILED are retried; READY is never re-checked")
}

/******************************************
 * Stage Two: The Download
 ******************************************/

// newDownload returns a service holding one WORKING attachment for a file, plus its fakes
func newDownload(t *testing.T, name string) (*StreamSource, streamSourceSession, model.Attachment, *fakeAdapter, *fakeAttachments, *fakeMedia) {

	t.Helper()

	adapter := &fakeAdapter{files: map[string]string{}}
	service, session, streamSource := newSyncService(adapter, &fakeStreamWriter{})

	// The StreamSource must be findable by its Stream, which is how the task finds its adapter
	session.collection.records = []model.StreamSource{streamSource}

	store := &fakeAttachments{}
	media := &fakeMedia{}
	service.attachmentService = store
	service.mediaServer = media

	attachment := newImportedAttachment(streamSource.StreamID, attachmentURL(name))
	require.NoError(t, store.Save(nil, &attachment, "test"))

	return service, session, attachment, adapter, store, media
}

// importAttachment runs the download task's service call for one attachment
func importAttachment(service *StreamSource, session streamSourceSession, attachment model.Attachment) error {
	return service.ImportAttachment(context.Background(), session, attachment.ObjectID, attachment.AttachmentID)
}

// TestStreamSourceAttachment_Download stores the file, marks the attachment READY with its sniffed
// type, and nudges the settings screen
func TestStreamSourceAttachment_Download(t *testing.T) {

	for name, file := range map[string]string{"a.png": pngFile, "a.mp4": mp4File, "a.pdf": pdfFile} {
		t.Run(name, func(t *testing.T) {

			service, session, attachment, adapter, store, media := newDownload(t, name)
			adapter.files[attachment.SourceURL] = file

			require.NoError(t, importAttachment(service, session, attachment))

			require.Equal(t, file, media.files[attachment.AttachmentID.Hex()], "the whole file, header included")
			require.Equal(t, model.AttachmentStatusReady, store.live()[0].Status)
			require.NotEmpty(t, store.live()[0].ContentType)

			nudges := session.publishedTasksNamed("PublishRealtimeMessage")
			require.Len(t, nudges, 1)
			require.Equal(t, attachment.ObjectID.Hex(), nudges[0].Arguments.GetString("objectId"))
			require.Equal(t, realtime.TopicStreamSourceUpdated, nudges[0].Arguments.GetInt("topic"))
		})
	}
}

// TestStreamSourceAttachment_CapFollowsTheKind confirms that a video is fetched with the larger cap
func TestStreamSourceAttachment_CapFollowsTheKind(t *testing.T) {

	service, session, attachment, adapter, _, _ := newDownload(t, "a.mp4")
	adapter.files[attachment.SourceURL] = mp4File

	require.NoError(t, importAttachment(service, session, attachment))
	require.Equal(t, int64(100<<20), adapter.fileMaxBytes)
}

// TestStreamSourceAttachment_NothingToDo confirms that a READY, deleted, or orphaned attachment
// downloads nothing
func TestStreamSourceAttachment_NothingToDo(t *testing.T) {

	t.Run("already ready", func(t *testing.T) {
		service, session, attachment, adapter, store, _ := newDownload(t, "a.png")
		store.records[0].Status = model.AttachmentStatusReady
		require.NoError(t, importAttachment(service, session, attachment))
		require.Empty(t, adapter.fileCalls)
	})

	t.Run("deleted", func(t *testing.T) {
		service, session, attachment, adapter, store, _ := newDownload(t, "a.png")
		store.records[0].DeleteDate = 1
		require.NoError(t, importAttachment(service, session, attachment))
		require.Empty(t, adapter.fileCalls)
	})

	t.Run("no source", func(t *testing.T) {
		service, session, attachment, adapter, _, _ := newDownload(t, "a.png")
		session.collection.records = nil
		require.NoError(t, importAttachment(service, session, attachment))
		require.Empty(t, adapter.fileCalls)
	})
}

// TestStreamSourceAttachment_AuthorErrorsFail confirms that a failure only the author can fix
// marks the attachment FAILED and stores nothing
func TestStreamSourceAttachment_AuthorErrorsFail(t *testing.T) {

	tests := map[string]struct {
		name string
		file string
	}{
		"text named .png":       {"a.png", "just some words, not an image"},
		"svg named .png":        {"a.png", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`},
		"image named .mp4":      {"a.mp4", pngFile},
		"image named .pdf":      {"a.pdf", pngFile},
		"missing at the origin": {"a.png", ""},
	}

	for label, test := range tests {
		t.Run(label, func(t *testing.T) {

			service, session, attachment, adapter, store, media := newDownload(t, test.name)

			if test.file != "" {
				adapter.files[attachment.SourceURL] = test.file
			}

			err := importAttachment(service, session, attachment)

			require.True(t, derp.IsClientError(err), "the author's mistake is not retried")
			require.Equal(t, model.AttachmentStatusFailed, store.live()[0].Status)
			require.Empty(t, media.files)
		})
	}
}

// TestStreamSourceAttachment_RetryableErrorsStayWorking confirms that an error the queue can retry
// leaves the attachment WORKING, so the settings screen does not read it as broken
func TestStreamSourceAttachment_RetryableErrorsStayWorking(t *testing.T) {

	tests := map[string]error{
		"server error":      derp.Internal("test", "Origin failed"),
		"too many requests": derp.NewHTTPError(nil, &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"60"}}}),
	}

	for label, fileError := range tests {
		t.Run(label, func(t *testing.T) {

			service, session, attachment, adapter, store, _ := newDownload(t, "a.png")
			adapter.fileError = fileError

			require.Error(t, importAttachment(service, session, attachment))
			require.Equal(t, model.AttachmentStatusWorking, store.live()[0].Status)
		})
	}
}

// TestStreamSourceAttachment_FailedWriteIsRemoved confirms that a file the MediaServer could not
// finish writing is deleted, so a partial file is never served as the whole
func TestStreamSourceAttachment_FailedWriteIsRemoved(t *testing.T) {

	service, session, attachment, adapter, store, media := newDownload(t, "a.png")
	adapter.files[attachment.SourceURL] = pngFile
	media.putError = derp.Internal("test", "Disk full")

	require.Error(t, importAttachment(service, session, attachment))
	require.Equal(t, []string{attachment.AttachmentID.Hex()}, media.deleted)
	require.Equal(t, model.AttachmentStatusWorking, store.live()[0].Status, "a server fault is retried")
}

// TestStreamSourceAttachment_SmallFile confirms that a file shorter than the sniffing header is
// read and stored whole
func TestStreamSourceAttachment_SmallFile(t *testing.T) {

	service, session, attachment, adapter, _, media := newDownload(t, "a.pdf")
	adapter.files[attachment.SourceURL] = pdfFile

	require.Less(t, len(pdfFile), 512)
	require.NoError(t, importAttachment(service, session, attachment))
	require.Equal(t, pdfFile, media.files[attachment.AttachmentID.Hex()])
}

// TestStreamSourceAttachment_Task confirms the queue task's name, arguments, and signature
func TestStreamSourceAttachment_Task(t *testing.T) {

	service, session := newStreamSourceService()
	service.hostname = "example.com"

	streamID := primitive.NewObjectID()
	attachmentID := primitive.NewObjectID()

	service.PublishAttachmentTask(session, streamID, attachmentID)

	tasks := session.publishedTasksNamed(TaskSyncStreamSourceAttachment)
	require.Len(t, tasks, 1)
	require.Equal(t, mapof.Any{"hostname": "example.com", "streamId": streamID.Hex(), "attachmentId": attachmentID.Hex()}, tasks[0].Arguments)
	require.Equal(t, "StreamSourceAttachment:"+attachmentID.Hex(), tasks[0].Signature)
}

// TestNewImportedAttachment_PassesTheSchema confirms that the record stage one builds, in each state
// it can reach, is one the real Attachment service will save.  The fakes above skip validation.
func TestNewImportedAttachment_PassesTheSchema(t *testing.T) {

	attachmentService := NewAttachment()

	for _, status := range []string{model.AttachmentStatusWorking, model.AttachmentStatusReady, model.AttachmentStatusFailed} {

		attachment := newImportedAttachment(primitive.NewObjectID(), attachmentURL("flow chart.png"))
		attachment.Status = status

		_, err := attachmentService.Schema().Validate(&attachment)
		require.NoError(t, err, "status %q", status)
	}
}

// TestAttachmentFilename returns the unescaped filename at the end of an address
func TestAttachmentFilename(t *testing.T) {
	require.Equal(t, "flow chart.png", attachmentFilename("https://raw.example.com/docs/attachments/flow%20chart.png?raw=1"))
	require.Equal(t, "", attachmentFilename("https://bad host/%zz"))
}
