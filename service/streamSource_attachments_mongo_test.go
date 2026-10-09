package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/content"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/mediaserver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamSourceAttachments_EndToEnd runs both stages against a real database, the real
// Attachment service, a real MediaServer, and a real origin over HTTP.  The unit tests use fakes
// that skip the database's own filters and the BSON encoding of SourceURL, which this does not.
// It skips when no local replica set is reachable.
func TestStreamSourceAttachments_EndToEnd(t *testing.T) {

	server, _ := newReplicaSetSession(t)

	// An origin holding one page and the two files it links to
	page := "# Guide\n\n![Flow](attachments/flow.png)\n\n![Demo](attachments/demo.mp4)\n"

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/docs/guide.md":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(page))
		case "/docs/attachments/flow.png":
			_, _ = w.Write([]byte(pngFile))
		case "/docs/attachments/demo.mp4":
			_, _ = w.Write([]byte(mp4File))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(origin.Close)

	// A session whose published tasks are spooled, so the test can run the downloads itself
	tasks := postcommit.NewTasks()
	session, err := server.Session(postcommit.WithContext(context.Background(), tasks))
	require.NoError(t, err)
	t.Cleanup(session.Close)

	// The real Attachment service and MediaServer, over an in-memory filesystem
	originals := afero.NewMemMapFs()
	media := mediaserver.New(originals, afero.NewMemMapFs(), mediaserver.NewWorkingDirectory(t.TempDir(), time.Minute, 10))

	attachmentService := NewAttachment()
	attachmentService.host = "https://site.example"
	attachmentService.mediaServer = media

	writer := &fakeStreamWriter{}
	service := NewStreamSource()
	service.streamService = writer
	service.attachmentService = &attachmentService
	service.mediaServer = media
	service.contentService = newTestContentService()
	service.adapters = map[string]content.Adapter{model.StreamSourceMethodHTTPS: content.NewHTTPS(true)}

	// The StreamSource record, stored so that the download can find its adapter
	streamSource := model.NewStreamSource()
	streamSource.StreamID = primitive.NewObjectID()
	streamSource.URL = origin.URL + "/docs/guide.md"
	writer.stream.StreamID = streamSource.StreamID
	require.NoError(t, service.collection(session).Save(&streamSource, "test"))

	// runDownloads runs every queued download, as the queue would, and returns how many ran
	runDownloads := func() int {
		count := 0
		for _, task := range tasks.Drain() {
			if task.Name != TaskSyncStreamSourceAttachment {
				continue
			}
			attachmentID, err := primitive.ObjectIDFromHex(task.Arguments.GetString("attachmentId"))
			require.NoError(t, err)
			require.NoError(t, service.ImportAttachment(context.Background(), session, streamSource.StreamID, attachmentID))
			count++
		}
		return count
	}

	// Stage one: the sync creates two WORKING attachments and rewrites the page
	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	imported, err := attachmentService.QueryByCategory(session, model.AttachmentObjectTypeStream, streamSource.StreamID, model.AttachmentCategoryStreamSource)
	require.NoError(t, err)
	require.Len(t, imported, 2)
	require.Equal(t, origin.URL+"/docs/attachments/flow.png", imported[0].SourceURL, "SourceURL survives the round trip through BSON")
	require.Equal(t, model.AttachmentStatusWorking, imported[0].Status)

	html := writer.saved[0].Content.HTML
	require.Contains(t, html, `src="https://site.example/`+streamSource.StreamID.Hex()+`/attachments/`+imported[0].AttachmentID.Hex()+`"`)
	require.Contains(t, html, `<video controls src="https://site.example/`+streamSource.StreamID.Hex()+`/attachments/`+imported[1].AttachmentID.Hex()+`">Demo</video>`)

	// Stage two: each download stores its file and marks it READY
	require.Equal(t, 2, runDownloads())

	imported, err = attachmentService.QueryByCategory(session, model.AttachmentObjectTypeStream, streamSource.StreamID, model.AttachmentCategoryStreamSource)
	require.NoError(t, err)

	for _, attachment := range imported {
		require.Equal(t, model.AttachmentStatusReady, attachment.Status)

		stored, err := afero.ReadFile(originals, attachment.AttachmentID.Hex())
		require.NoError(t, err)
		require.NotEmpty(t, stored)
	}

	require.Equal(t, "image/png", imported[0].ContentType)
	require.Equal(t, "video/mp4", imported[1].ContentType)

	// A sync with nothing new re-queues nothing, because every file is READY
	require.NoError(t, service.Sync(context.Background(), session, &streamSource))
	require.Zero(t, runDownloads())

	// The page drops the video: its attachment and its file are deleted, and the image is kept
	page = "# Guide\n\n![Flow](attachments/flow.png)\n"
	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	remaining, err := attachmentService.QueryByCategory(session, model.AttachmentObjectTypeStream, streamSource.StreamID, model.AttachmentCategoryStreamSource)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	require.Equal(t, imported[0].AttachmentID, remaining[0].AttachmentID)

	exists, err := afero.Exists(originals, imported[1].AttachmentID.Hex())
	require.NoError(t, err)
	require.False(t, exists, "the video's file is removed with its attachment")
}
