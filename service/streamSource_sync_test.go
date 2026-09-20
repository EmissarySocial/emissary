package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/content"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/davidscottmills/goeditorjs"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * In-Memory Fakes
 ******************************************/

// fakeAdapter stands in for a remote source, and counts the round trips a sync makes
type fakeAdapter struct {
	version      string
	versionError error
	item         content.Item
	fetchError   error
	versionCalls int
	fetchCalls   int
	ifNoneMatch  string // the stored version that Version() was asked about
}

// Protocol implements the content.Adapter interface
func (adapter *fakeAdapter) Protocol() string {
	return model.StreamSourceMethodHTTPS
}

// Version implements the content.Adapter interface
func (adapter *fakeAdapter) Version(_ context.Context, source model.StreamSource) (string, error) {
	adapter.versionCalls++
	adapter.ifNoneMatch = source.Version
	return adapter.version, adapter.versionError
}

// Fetch implements the content.Adapter interface
func (adapter *fakeAdapter) Fetch(_ context.Context, _ model.StreamSource, _ string) (content.Item, error) {
	adapter.fetchCalls++
	return adapter.item, adapter.fetchError
}

// Subscribe implements the content.Adapter interface
func (adapter *fakeAdapter) Subscribe(_ context.Context, _ model.StreamSource, _ string) error {
	return derp.NotImplemented("test", "unused")
}

// fakeStreamWriter stands in for the Stream service, which a unit test cannot assemble
type fakeStreamWriter struct {
	stream        model.Stream
	loadError     error
	validateError error
	validated     []string // every token passed to ValidateToken, in order
	saved         []model.Stream
	saveError     error
}

// LoadByID implements the streamWriter interface
func (writer *fakeStreamWriter) LoadByID(_ data.Session, _ primitive.ObjectID, result *model.Stream) error {

	if writer.loadError != nil {
		return writer.loadError
	}

	*result = writer.stream
	return nil
}

// ValidateToken implements the streamWriter interface
func (writer *fakeStreamWriter) ValidateToken(_ data.Session, _ primitive.ObjectID, token string) error {
	writer.validated = append(writer.validated, token)
	return writer.validateError
}

// Save implements the streamWriter interface
func (writer *fakeStreamWriter) Save(_ data.Session, stream *model.Stream, _ string) error {

	if writer.saveError != nil {
		return writer.saveError
	}

	writer.saved = append(writer.saved, *stream)
	return nil
}

/******************************************
 * Test Helpers
 ******************************************/

// newSyncService returns a StreamSource service wired to fakes, plus the record it will synchronize
func newSyncService(adapter *fakeAdapter, writer *fakeStreamWriter) (*StreamSource, streamSourceSession, model.StreamSource) {

	streamSource := validStreamSource()

	service, session := newStreamSourceService(streamSource)
	service.streamService = writer
	service.contentService = newTestContentService()
	service.adapters = map[string]content.Adapter{
		model.StreamSourceMethodHTTPS: adapter,
	}

	writer.stream.StreamID = streamSource.StreamID

	return service, session, streamSource
}

// newTestContentService returns the REAL Content service, so that a test proves content.html was
// actually rendered rather than that a fake was called
func newTestContentService() *Content {
	result := NewContent(goeditorjs.NewHTMLEngine())
	return &result
}

// markdownItem builds the Item that a fake adapter will return
func markdownItem(t *testing.T, file string) content.Item {
	t.Helper()
	item, err := content.NewItem(model.ContentFormatMarkdown, []byte(file))
	require.NoError(t, err)
	return item
}

/******************************************
 * The Version Check
 ******************************************/

// TestStreamSourceSync_UnchangedVersionSkipsTheFetch confirms that a 304 ends a sync after one
// round trip, and never touches the Stream
func TestStreamSourceSync_UnchangedVersionSkipsTheFetch(t *testing.T) {

	adapter := &fakeAdapter{version: `"abc123"`}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	streamSource.Version = `"abc123"`

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Equal(t, 1, adapter.versionCalls)
	require.Equal(t, 0, adapter.fetchCalls, "a source that has not changed is never downloaded")
	require.Empty(t, writer.saved, "an unchanged source never calls Stream.Save")
	require.Equal(t, `"abc123"`, adapter.ifNoneMatch, "the stored version is what makes the request conditional")
}

// TestStreamSourceSync_EmptyVersionAlwaysFetches pins the rule that separates "nothing changed"
// from "this origin cannot tell me".  cgit and SourceHut answer no validator at all, so an empty
// version that ended a sync would freeze those sources at whatever they held on the first run.
func TestStreamSourceSync_EmptyVersionAlwaysFetches(t *testing.T) {

	adapter := &fakeAdapter{version: "", item: markdownItem(t, "# Hello")}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	streamSource.Version = ""

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Equal(t, 1, adapter.fetchCalls, "an origin with no validator is read on every ping")
	require.Len(t, writer.saved, 1)
}

// TestStreamSourceSync_NewValidatorSameBytes records the new validator without republishing the
// Stream.  Stream.Save federates and notifies, so it must not run for identical content.
func TestStreamSourceSync_NewValidatorSameBytes(t *testing.T) {

	item := markdownItem(t, "# Hello")
	adapter := &fakeAdapter{version: `"new"`, item: item}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	streamSource.Version = `"old"`
	streamSource.ContentHash = item.Hash

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Equal(t, 1, adapter.fetchCalls)
	require.Empty(t, writer.saved, "identical bytes never reach Stream.Save")
	require.Equal(t, `"new"`, streamSource.Version, "the new validator is stored, so the next ping is a 304")
	require.Equal(t, item.Hash, streamSource.ContentHash)
}

/******************************************
 * Writing to the Stream
 ******************************************/

// TestStreamSourceSync_WritesRenderedHTML pins C1: content.html is the only body a remote reader
// ever sees, so a sync that stored raw Markdown alone would publish an empty page to the fediverse
func TestStreamSourceSync_WritesRenderedHTML(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "# Title\n\nSome *emphasis* here.")}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, writer.saved, 1)
	saved := writer.saved[0]

	require.Equal(t, model.ContentFormatMarkdown, saved.Content.Format)
	require.Contains(t, saved.Content.Raw, "# Title")
	require.Contains(t, saved.Content.HTML, "<h1", "the body must be rendered, not stored raw")
	require.Contains(t, saved.Content.HTML, "<em>emphasis</em>")
}

// TestStreamSourceSync_RecordsWhatItApplied confirms that a successful sync stores the version and
// hash that let the NEXT ping skip the work
func TestStreamSourceSync_RecordsWhatItApplied(t *testing.T) {

	item := markdownItem(t, "# Hello")
	adapter := &fakeAdapter{version: `"v1"`, item: item}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Equal(t, `"v1"`, streamSource.Version)
	require.Equal(t, item.Hash, streamSource.ContentHash)
	require.NotZero(t, streamSource.LastSynced)

	// Every exit from a sync saves the record, so an operator can see that it ran
	require.NotEmpty(t, session.collection.saved)
}

// TestStreamSourceSync_EveryExitRecordsLastSynced confirms that a sync which changes nothing still
// leaves evidence that it ran.  Under D11 that timestamp is how an operator tells a working
// webhook from one nobody ever installed.
func TestStreamSourceSync_EveryExitRecordsLastSynced(t *testing.T) {

	item := markdownItem(t, "# Hello")

	// notModified, sameBytes, and changed are the three ways a sync can end
	notModified := func() (*fakeAdapter, model.StreamSource) {
		source := validStreamSource()
		source.Version = `"same"`
		return &fakeAdapter{version: `"same"`}, source
	}

	sameBytes := func() (*fakeAdapter, model.StreamSource) {
		source := validStreamSource()
		source.ContentHash = item.Hash
		return &fakeAdapter{version: `"new"`, item: item}, source
	}

	changed := func() (*fakeAdapter, model.StreamSource) {
		return &fakeAdapter{version: `"new"`, item: item}, validStreamSource()
	}

	for name, build := range map[string]func() (*fakeAdapter, model.StreamSource){
		"not modified": notModified,
		"same bytes":   sameBytes,
		"changed":      changed,
	} {
		t.Run(name, func(t *testing.T) {

			adapter, streamSource := build()
			writer := &fakeStreamWriter{}
			service, session, _ := newSyncService(adapter, writer)
			writer.stream.StreamID = streamSource.StreamID

			require.NoError(t, service.Sync(context.Background(), session, &streamSource))

			require.NotZero(t, streamSource.LastSynced)
			require.Len(t, session.collection.saved, 1, "a sync saves its record exactly once")
			require.NotZero(t, session.collection.saved[0].LastSynced)
		})
	}
}

/******************************************
 * Front Matter
 ******************************************/

// TestStreamSourceSync_FrontMatter applies the values a source carries
func TestStreamSourceSync_FrontMatter(t *testing.T) {

	file := "---\ntitle: Getting Started\nsummary: How to begin\nrank: 7\n---\n\n# Body"
	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, file)}
	writer := &fakeStreamWriter{}
	writer.stream.Label = "Old Label"
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, writer.saved, 1)
	require.Equal(t, "Getting Started", writer.saved[0].Label)
	require.Equal(t, "How to begin", writer.saved[0].Summary)
	require.Equal(t, 7, writer.saved[0].Rank)
	require.NotContains(t, writer.saved[0].Content.Raw, "title:", "front matter is metadata, not body")
}

// TestStreamSourceSync_NoFrontMatterChangesNothing pins D1: a source with no front matter replaces
// the body and leaves every other Stream field exactly as it was
func TestStreamSourceSync_NoFrontMatterChangesNothing(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "# Just a heading")}
	writer := &fakeStreamWriter{}
	writer.stream.Label = "Chosen By A Human"
	writer.stream.Summary = "Also chosen by a human"
	writer.stream.Rank = 3
	writer.stream.Token = "human-token"
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, writer.saved, 1)
	require.Equal(t, "Chosen By A Human", writer.saved[0].Label)
	require.Equal(t, "Also chosen by a human", writer.saved[0].Summary)
	require.Equal(t, 3, writer.saved[0].Rank)
	require.Equal(t, "human-token", writer.saved[0].Token)
	require.Empty(t, writer.validated, "no slug means ValidateToken is never consulted")
}

// TestStreamSourceSync_PartialFrontMatterLeavesTheRest confirms that front matter is applied key by
// key.  A file that names only a title must not blank out a summary somebody wrote by hand.
func TestStreamSourceSync_PartialFrontMatterLeavesTheRest(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "---\ntitle: Only A Title\n---\n\nBody")}
	writer := &fakeStreamWriter{}
	writer.stream.Summary = "Written by a human"
	writer.stream.Rank = 4
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, writer.saved, 1)
	require.Equal(t, "Only A Title", writer.saved[0].Label)
	require.Equal(t, "Written by a human", writer.saved[0].Summary)
	require.Equal(t, 4, writer.saved[0].Rank)
}

// TestStreamSourceSync_SlugIsValidated pins D2 and C5: a front matter slug becomes the Stream's
// token only after ValidateToken accepts it
func TestStreamSourceSync_SlugIsValidated(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "---\nslug: getting-started\n---\n\nBody")}
	writer := &fakeStreamWriter{}
	writer.stream.Token = "old-token"
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Equal(t, []string{"getting-started"}, writer.validated)
	require.Len(t, writer.saved, 1)
	require.Equal(t, "getting-started", writer.saved[0].Token)
}

// TestStreamSourceSync_RejectedSlugFailsTheWholeSync confirms that a token collision stops
// everything.  Publishing the new body under the old URL would hide the conflict until somebody
// followed a link that no longer worked.
func TestStreamSourceSync_RejectedSlugFailsTheWholeSync(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "---\nslug: taken\n---\n\nBody")}
	writer := &fakeStreamWriter{validateError: derp.BadRequest("test", "This token is already in use")}
	service, session, streamSource := newSyncService(adapter, writer)

	err := service.Sync(context.Background(), session, &streamSource)

	require.Error(t, err)
	require.True(t, derp.IsClientError(err), "an author's mistake is a 4xx, not a defect: %v", err)
	require.Empty(t, writer.saved, "a rejected slug never saves the Stream")
	require.Empty(t, streamSource.ContentHash, "a failed sync records nothing, so the retry starts clean")
}

// TestStreamSourceSync_EmptySlugIsRefused confirms that a blank slug reaches ValidateToken rather
// than being quietly skipped, so the author is told their front matter is wrong
func TestStreamSourceSync_EmptySlugIsRefused(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "---\nslug: \"\"\n---\n\nBody")}
	writer := &fakeStreamWriter{validateError: derp.BadRequest("test", "Token must be at least 3 characters")}
	service, session, streamSource := newSyncService(adapter, writer)

	require.Error(t, service.Sync(context.Background(), session, &streamSource))
	require.Equal(t, []string{""}, writer.validated)
}

// TestStreamSourceSync_NonNumericRankIsIgnored confirms that an unreadable rank leaves the Stream's
// own sort order alone, instead of silently moving the page to the front
func TestStreamSourceSync_NonNumericRankIsIgnored(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`, item: markdownItem(t, "---\nrank: soon\n---\n\nBody")}
	writer := &fakeStreamWriter{}
	writer.stream.Rank = 12
	service, session, streamSource := newSyncService(adapter, writer)

	require.NoError(t, service.Sync(context.Background(), session, &streamSource))

	require.Len(t, writer.saved, 1)
	require.Equal(t, 12, writer.saved[0].Rank)
}

/******************************************
 * Failure Paths
 ******************************************/

// TestStreamSourceSync_Failures confirms that every failure stops the sync, and that an author's
// mistake is never filed as one of Emissary's defects
func TestStreamSourceSync_Failures(t *testing.T) {

	item := markdownItem(t, "# Hello")

	fails := func(name string, isClientError bool, build func() (*fakeAdapter, *fakeStreamWriter)) {
		t.Run(name, func(t *testing.T) {

			adapter, writer := build()
			service, session, streamSource := newSyncService(adapter, writer)

			err := service.Sync(context.Background(), session, &streamSource)

			require.Error(t, err)
			require.Equal(t, isClientError, derp.IsClientError(err), "got %v", err)
			require.Empty(t, writer.saved)
			require.Empty(t, streamSource.Version, "a failed sync records nothing")
		})
	}

	fails("origin refuses the address", true, func() (*fakeAdapter, *fakeStreamWriter) {
		return &fakeAdapter{versionError: derp.NotFound("test", "Source not found")}, &fakeStreamWriter{}
	})

	fails("origin server is broken", false, func() (*fakeAdapter, *fakeStreamWriter) {
		return &fakeAdapter{versionError: derp.Internal("test", "Source server failed")}, &fakeStreamWriter{}
	})

	fails("content cannot be read", true, func() (*fakeAdapter, *fakeStreamWriter) {
		return &fakeAdapter{version: `"v1"`, fetchError: derp.BadRequest("test", "Source is not Markdown")}, &fakeStreamWriter{}
	})

	fails("stream is gone", true, func() (*fakeAdapter, *fakeStreamWriter) {
		return &fakeAdapter{version: `"v1"`, item: item}, &fakeStreamWriter{loadError: derp.NotFound("test", "Stream not found")}
	})

	fails("stream will not save", false, func() (*fakeAdapter, *fakeStreamWriter) {
		return &fakeAdapter{version: `"v1"`, item: item}, &fakeStreamWriter{saveError: derp.Internal("test", "Saving Stream")}
	})
}

// TestStreamSourceSync_UnknownMethod confirms that a record naming an Adapter that does not exist
// is a defect in the adapter table, not something a retry can repair
func TestStreamSourceSync_UnknownMethod(t *testing.T) {

	adapter := &fakeAdapter{version: `"v1"`}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	streamSource.Method = "CARRIER-PIGEON"

	err := service.Sync(context.Background(), session, &streamSource)

	require.Error(t, err)
	require.Equal(t, 500, derp.ErrorCode(err))
	require.Equal(t, 0, adapter.versionCalls)
}

// TestStreamSourceSync_NeverEchoesAPassword confirms that no part of a sync repeats an address that
// carries credentials, however the sync fails
func TestStreamSourceSync_NeverEchoesAPassword(t *testing.T) {

	adapter := &fakeAdapter{versionError: derp.NotFound("test", "Source not found")}
	writer := &fakeStreamWriter{}
	service, session, streamSource := newSyncService(adapter, writer)

	streamSource.URL = "https://user:hunter2@example.com/README.md"

	err := service.Sync(context.Background(), session, &streamSource)

	require.Error(t, err)

	encoded, marshalErr := json.Marshal(err)
	require.NoError(t, marshalErr)
	require.NotContains(t, string(encoded), "hunter2")
}

/******************************************
 * Service Wiring
 ******************************************/

// TestStreamSource_RefreshWiresEveryDependency confirms that Refresh resolves everything a sync
// reaches for.  Nothing else in the suite runs Refresh, so a dependency left nil here would first
// appear as a panic in production, on the first webhook ping.
func TestStreamSource_RefreshWiresEveryDependency(t *testing.T) {

	contentService := NewContent(goeditorjs.NewHTMLEngine())

	factory := Factory{
		contentService: &contentService,
		serverFactory:  &lifecycleServerFactory{},
	}

	factory.config.Hostname = "example.com"
	factory.streamSourceService.Refresh(&factory)

	service := factory.StreamSource()

	streamService, isStreamService := service.streamService.(*Stream)
	require.True(t, isStreamService, "the Stream service must satisfy streamWriter")
	require.Same(t, factory.Stream(), streamService)

	require.Same(t, &contentService, service.contentService)
	require.Equal(t, "example.com", service.hostname, "a task cannot find its Domain without this")

	// The Adapter table is keyed by the Method that a record stores, so a key which drifted from
	// the model constant would make every record unreadable with no compile-time complaint
	adapter, err := service.adapterFor(model.StreamSourceMethodHTTPS)
	require.NoError(t, err)
	require.Equal(t, model.StreamSourceMethodHTTPS, adapter.Protocol())
}
