package service

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/content"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

/******************************************
 * StreamSource Synchronization
 *
 * One pass of the pipeline in section 5 of
 * GIT-MARKDOWN-TO-STREAM-CONTENT.md: ask the
 * origin what it holds, read it when it has
 * changed, and push it into a Stream.
 ******************************************/

// Sync copies the remote source's current content into the Stream that this record populates
func (service *StreamSource) Sync(ctx context.Context, session data.Session, streamSource *model.StreamSource) error {

	const location = "service.StreamSource.Sync"

	// Find the Adapter that reads this kind of source
	adapter, err := service.adapterFor(streamSource.Method)

	if err != nil {
		return derp.Wrap(err, location, "Unable to read this source", streamSource.StreamSourceID)
	}

	// Recorded now, because it marks when this attempt BEGAN, and every exit below saves it
	streamSource.LastSynced = time.Now().Unix()

	// Ask the origin for its validator, sending the stored one so that an unchanged source can
	// answer 304 with no body at all
	version, err := adapter.Version(ctx, *streamSource)

	if err != nil {
		return derp.Wrap(err, location, "Reading source version", streamSource.StreamSourceID)
	}

	// RULE: An EMPTY version means the origin offers no validator, not that nothing changed, so
	// it can never end a sync.  Treating it as "unchanged" would freeze every cgit and SourceHut
	// source at the content it was created with.
	if (version != "") && (version == streamSource.Version) {
		return service.saveSyncState(session, streamSource, "Checked: unchanged")
	}

	// Read the file itself
	item, err := adapter.Fetch(ctx, *streamSource, version)

	if err != nil {
		return derp.Wrap(err, location, "Reading source content", streamSource.StreamSourceID)
	}

	// RULE: A new validator over identical bytes updates this record and stops.  Stream.Save
	// federates, notifies, and rewrites the whole document, so an unchanged page must not call it.
	if item.Hash == streamSource.ContentHash {
		streamSource.Version = version
		return service.saveSyncState(session, streamSource, "Checked: unchanged")
	}

	// Push the content into the Stream
	if err := service.applyToStream(session, streamSource, item); err != nil {
		return derp.Wrap(err, location, "Applying content to Stream", streamSource.StreamSourceID)
	}

	// Record what was applied, so that the next ping can skip it
	streamSource.Version = version
	streamSource.ContentHash = item.Hash

	return service.saveSyncState(session, streamSource, "Synchronized")
}

/******************************************
 * Helper Methods
 ******************************************/

// applyToStream writes a remote Item into the Stream that a StreamSource record populates
func (service *StreamSource) applyToStream(session data.Session, streamSource *model.StreamSource, item content.Item) error {

	const location = "service.StreamSource.applyToStream"

	// Load the Stream that this record populates
	stream := model.NewStream()

	if err := service.streamService.LoadByID(session, streamSource.StreamID, &stream); err != nil {
		return derp.Wrap(err, location, "Loading Stream", streamSource.StreamID)
	}

	// Apply front matter values, when the source carries any
	if err := service.applyFrontMatter(session, &stream, item.Meta); err != nil {
		return derp.Wrap(err, location, "Applying front matter", streamSource.StreamSourceID)
	}

	// RULE: Content is built by the Content service, which renders the HTML that every remote
	// reader sees.  Writing the raw value alone would publish an empty body to the fediverse.
	stream.Content = service.contentService.New(item.Format, string(item.Source))

	// Save the Stream, which does whatever it would do for a human edit: no state change, no
	// settings change, and federation decided by this Stream's own configuration.
	if err := service.streamService.Save(session, &stream, "Synced from "+streamSource.URL); err != nil {
		return derp.Wrap(err, location, "Saving Stream", streamSource.StreamID)
	}

	return nil
}

// applyFrontMatter copies a source's front matter onto the Stream it populates.  A value that the
// front matter does not carry leaves its field untouched.
func (service *StreamSource) applyFrontMatter(session data.Session, stream *model.Stream, meta mapof.Any) error {

	const location = "service.StreamSource.applyFrontMatter"

	// A source with no front matter changes no Stream field at all
	if len(meta) == 0 {
		return nil
	}

	if value, exists := frontMatterValue(meta, "title", "label"); exists {
		stream.Label = convert.String(value)
	}

	if value, exists := frontMatterValue(meta, "summary", "description"); exists {
		stream.Summary = convert.String(value)
	}

	// A value that is not a number leaves the rank alone, instead of silently sorting the page first
	if value, exists := frontMatterValue(meta, "rank", "weight"); exists {
		if rank, ok := convert.IntOk(value, stream.Rank); ok {
			stream.Rank = rank
		}
	}

	value, exists := frontMatterValue(meta, "slug")

	if !exists {
		return nil
	}

	// RULE: A token names this Stream everywhere on the Domain, so a rejected slug fails the whole
	// sync.  Publishing the new body under the old URL would hide the conflict until someone
	// followed a link that no longer worked.
	token := convert.String(value)

	if err := service.streamService.ValidateToken(session, stream.StreamID, token); err != nil {
		return derp.Wrap(err, location, "Front matter 'slug' cannot be used", token)
	}

	stream.Token = token

	return nil
}

// saveSyncState writes a record's synchronization bookkeeping, and nothing else
func (service *StreamSource) saveSyncState(session data.Session, streamSource *model.StreamSource, note string) error {

	const location = "service.StreamSource.saveSyncState"

	// Bookkeeping skips Save's validation, which a version, a hash, and a timestamp cannot violate
	if err := service.collection(session).Save(streamSource, note); err != nil {
		return derp.Wrap(err, location, "Saving StreamSource", streamSource.StreamSourceID)
	}

	// Another day, another sync
	return nil
}

// frontMatterValue returns the first of the provided names that the front matter carries
func frontMatterValue(meta mapof.Any, names ...string) (any, bool) {

	for _, name := range names {
		if value, exists := meta[name]; exists {
			return value, true
		}
	}

	return nil, false
}
