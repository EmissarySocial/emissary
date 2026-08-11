package service

import (
	"crypto"
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * ActivityPub API
 ******************************************/

// JSONLDGetter returns a new JSONLDGetter for the provided stream
func (service *Stream) JSONLDGetter(session data.Session, stream *model.Stream) StreamJSONLDGetter {
	return NewStreamJSONLDGetter(session, service, stream)
}

func (service *Stream) Activity(session data.Session, stream *model.Stream) streams.Document {
	// Create a new ActivityPub Document for this Stream
	return streams.NewDocument(service.JSONLD(session, stream))
}

// GetJSONLD returns a map document that conforms to the ActivityStreams 2.0 spec.
// This map will still need to be marshalled into JSON
func (service *Stream) JSONLD(session data.Session, stream *model.Stream) mapof.Any {

	const location = "service.Stream.JSONLD"

	// Load the Template for this Stream.  It provides the hashtag URLs and the "social mapping" below.
	template, templateErr := service.templateService.Load(stream.TemplateID)

	if templateErr != nil {
		derp.Report(derp.Wrap(templateErr, location, "Loading Template", stream.TemplateID))
	}

	result := mapof.Any{
		vocab.AtContext:       sliceof.Any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyID:      stream.ActivityPubURL(),
		vocab.PropertyType:    stream.SocialRole,
		vocab.PropertyURL:     stream.URL,
		vocab.PropertyReplies: stream.ActivityPubRepliesURL(),
	}

	// PublishDate carries two sentinels that are not real dates: 0 (unpublished)
	// and math.MaxInt64 (not yet scheduled). FromUnixSeconds renders both as "",
	// so omit `published` rather than federating a 1970 or 12-digit-year timestamp.
	if published := datetime.FromUnixSeconds(stream.PublishDate); published != "" {
		result[vocab.PropertyPublished] = published
	}

	if stream.Label != "" {
		result[vocab.PropertyName] = stream.Label
	}

	/* REMOVED SUMMARY because this is used by Mastodon as a Content Warning
	if stream.Summary != "" {
		result[vocab.PropertySummary] = stream.Summary
	} */

	if stream.Content.HTML != "" {
		result[vocab.PropertyContent] = stream.Content.HTML
	}

	if stream.Context != "" {
		result[vocab.PropertyContext] = stream.Context
	}

	if stream.InReplyTo != "" {
		result[vocab.PropertyInReplyTo] = stream.InReplyTo
	}

	if stream.AttributedTo.NotEmpty() {
		result[vocab.PropertyAttributedTo] = stream.AttributedTo.ProfileURL
	}

	// The `tag` array carries BOTH #hashtags and @mentions, and is assigned exactly once, so
	// both must be collected before it is written.
	tags := make([]mapof.String, 0, len(stream.Hashtags)+len(stream.Mentions))

	for _, hashtag := range stream.Hashtags {
		tags = append(tags, service.HashtagAsJSONLD(template.TagURL, hashtag))
	}

	// RULE: Only RESOLVED @mentions are federated. An unresolved handle stays on the Stream so
	// the author can correct it, but a `Mention` tag with no `href` gives a receiving server
	// nothing to route to, so it is not published. (resolveMentions fills these in at publish.)
	mentioned := make([]string, 0, len(stream.Mentions))

	for _, mention := range stream.Mentions {

		if mention.NotResolved() {
			continue
		}

		tags = append(tags, mention.JSONLD())
		mentioned = append(mentioned, mention.Href)
	}

	if len(tags) > 0 {
		result[vocab.PropertyTag] = tags
	}

	// RULE: A mentioned actor must also be ADDRESSED. Mastodon resolves mentions from the `tag`
	// array, but delivery is driven by addressing -- without this, a mentioned account that does
	// not already follow the author never receives the post at all. Outbox.Publish delivers to
	// every addressee on top of the follower fan-out (see publishRecipients).
	//
	// This is a plain []string because service.Stream.publish_outbox type-asserts it as one when
	// it appends the in-reply-to author; a named slice type would silently fail that assertion.
	if len(mentioned) > 0 {
		result[vocab.PropertyCC] = mentioned
	}

	if stream.Location.NotZero() {
		result[vocab.PropertyLocation] = stream.Location.JSONLD()
	}

	// NOTE: According to Mastodon ActivityPub guide (https://docs.joinmastodon.org/spec/activitypub/)
	// putting as:public in the To field means that this mesage is public, and "listed"
	// putting as:public in the Cc field means that this message is public, but "unlisted"
	// and leaving as:public out entirely means that this message is "private" -- for whatever that's worth...

	if stream.DefaultAllowAnonymous() {
		result[vocab.PropertyTo] = []string{vocab.NamespacePublic}
	}

	// Custom behaviors for different stream types
	switch stream.SocialRole {

	case vocab.ObjectTypeAudio:

		// Size (in bytes)
		// Bitrate
		// Duration
		// Library (custom Funkwhale type)

		if attachments, err := service.attachmentService.QueryByCategory(session, model.AttachmentObjectTypeStream, stream.StreamID, vocab.ObjectTypeAudio); err == nil {
			link := make([]mapof.Any, 0, len(attachments))

			for _, attachment := range attachments {
				link = append(link, mapof.Any{
					vocab.PropertyType:      vocab.CoreTypeLink,
					vocab.PropertyHref:      stream.ActivityPubURL() + "/attachments/" + attachment.AttachmentID.Hex() + ".mpg",
					vocab.PropertyMediaType: "audio/mpeg",
					vocab.PropertyName:      first.String(attachment.Description, attachment.Label, attachment.Category),
				})
			}

			switch len(link) {
			case 0: // Do nothing
			case 1:
				result[vocab.PropertyURL] = link[0]
			default:
				result[vocab.PropertyURL] = link
			}
		}
	}

	// Include attachments for all types (including Audio)
	if attachments, err := service.attachmentService.QueryByObjectID(session, model.AttachmentObjectTypeStream, stream.StreamID); err == nil {

		attachmentJSON := make([]mapof.Any, 0, len(attachments))
		for _, attachment := range attachments {
			attachmentJSON = append(attachmentJSON, attachment.JSONLD())
		}

		result[vocab.PropertyAttachment] = attachmentJSON
	}

	// Try to apply the "social mapping" to the stream
	if templateErr == nil {
		result[vocab.PropertyType] = template.SocialRole
		if template.SocialRules.NotEmpty() {
			schma := service.activityStreamSchema()
			if err := template.SocialRules.Execute(schma, stream, schma, &result); err != nil {
				derp.Report(derp.Wrap(err, location, "Applying social rules to stream", stream.StreamID, template.SocialRules))
			}
		}
	}

	return result
}

// HashtagAsJSONLD returns a JSON-LD map document that represents a hashtag.
// The tagURL is the link prefix defined by the Stream's Template; it is made
// absolute because this document is read by other servers.  When the Template
// defines no tagURL, the hashtag is published without a link.
func (service *Stream) HashtagAsJSONLD(tagURL string, tag string) mapof.String {

	result := mapof.String{
		vocab.PropertyType: vocab.LinkTypeHashtag,
		vocab.PropertyName: "#" + tag,
	}

	// Include the link target when the Template defines one
	if href := model.HashtagURL(service.host, tagURL, tag); href != "" {
		result[vocab.PropertyHref] = href
	}

	return result
}

func (service *Stream) ActivityPubURL(streamID primitive.ObjectID) string {
	return service.host + "/" + streamID.Hex()
}

func (service *Stream) PublicKeyID(streamID primitive.ObjectID) string {
	return service.ActivityPubURL(streamID) + "#main-key"
}

func (service *Stream) PrivateKey(session data.Session, streamID primitive.ObjectID) (crypto.PrivateKey, error) {

	const location = "service.Stream.PrivateKey"

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByParentID(session, model.EncryptionKeyTypeStream, streamID, &encryptionKey); err != nil {
		return nil, derp.Wrap(err, location, "Loading encryption key", streamID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Extracting private key", encryptionKey)
	}

	// Success
	return privateKey, nil

}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided Stream.
func (service *Stream) ActivityPubActor(session data.Session, streamID primitive.ObjectID) (outbox.Actor, error) {

	const location = "service.Stream.ActivityPubActor"

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByParentID(session, model.EncryptionKeyTypeStream, streamID, &encryptionKey); err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Loading encryption key", streamID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Extracting private key", encryptionKey)
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActivityPubURL(streamID),
		privateKey,
		outbox.WithFollowers(service.RangeActivityPubFollowers(session, streamID)),
		outbox.WithClient(service.activityService.StreamClient(streamID)),
		outbox.WithAllowPrivateIPs(service.activityService.AllowPrivateIPs()),
	)

	return actor, nil
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided User.
func (service *Stream) RangeActivityPubFollowers(session data.Session, streamID primitive.ObjectID) iter.Seq[string] {

	return func(yield func(string) bool) {

		// Retrieve all Followers for this Stream
		followers := service.followerService.RangeActivityPubByType(session, model.FollowerTypeStream, streamID)

		for follower := range followers {
			if !yield(follower.Actor.ProfileURL) {
				return // Stop iterating if the yield function returns false
			}
		}
	}
}

func (service *Stream) activityStreamSchema() schema.Schema {

	return schema.New(
		schema.Object{
			Properties: schema.ElementMap{
				"@context": schema.Array{Items: schema.Any{}},
			},
			Wildcard: schema.Any{},
		},
	)
}

// Post updates the stream with a new Context Collection (if none already exists)
func (service *Stream) CalcContext(session data.Session, stream *model.Stream) error {

	const location = "service.Stream.CalcContext"

	// RULE: If the stream is not public, then don't create a context collection.
	// For the time being, these are only for public-facing posts.
	if !stream.DefaultAllow.IsAnonymous() {
		return nil
	}

	// RULE: If a context is already defined for this Stream, then keep it. Don't recalculate.
	if stream.Context != "" {
		return nil
	}

	// If this is a reply, then try to inherit a context from our ancestors
	if inReplyTo := stream.InReplyTo; inReplyTo != "" {

		// Create an ActivityStreams client based on the Stream author's permissions
		client := service.activityService.UserClient(stream.AttributedTo.UserID)

		// Scan no more than 5 documents UP the reply chain to inherit
		for range 5 {

			// Get an ActivityStreams client for this content, and load the document that is being replied to
			document, err := client.Load(inReplyTo)

			if err != nil {
				return derp.Wrap(err, location, "Loading InReplyTo document", inReplyTo)
			}

			// If this document has a context then use it and exit
			if context := document.Context(); context != "" {
				stream.Context = context
				return nil
			}

			// If this document is a reply, then keep looking UP the reply chain
			inReplyTo = document.InReplyTo().String()

			// If we have reached the top of the reply chain, then stop scanning.
			if inReplyTo == "" {
				break
			}
		}
	}

	// Fall through means: a) This is an original post (not a reply), or b) No ancestor supplied a context.
	// Let's create a new Context Collection for this Stream (and descendants)

	// Create a new Context Collection. ParentID + CollectionType are set so it participates
	// in the unique (parentId, collectionType) index and is addressable by later lookups.
	collection := model.NewCollection()
	collection.UserID = stream.AttributedTo.UserID
	collection.ParentID = stream.StreamID
	collection.ParentType = model.CollectionParentTypeStream
	collection.CollectionType = model.CollectionTypeContext
	collection.Read = sliceof.String{vocab.NamespacePublic}  // <- this will need to be updated when we add support for non-public streams.
	collection.Write = sliceof.String{vocab.NamespacePublic} // <- this will need to be updated when we add support for non-public streams.

	// Set the Stream to use the Context Collection
	stream.Context = service.collectionService.ActivityPubURL(collection.UserID, collection.CollectionID)

	// Save the Context Collection
	if err := service.collectionService.Save(session, &collection, "Created for new Stream context"); err != nil {
		return derp.Wrap(err, location, "Saving context collection", stream)
	}

	// Add this Stream to the Context Collection
	if err := service.collectionService.AddItem(session, &collection, stream.ActivityPubURL(), stream.InReplyTo); err != nil {
		return derp.Wrap(err, location, "Adding Stream to context collection", stream)
	}

	// Success!
	return nil
}

// AddReply records replyURL in the JIT Replies collection of the local Stream that
// inReplyToURL points to. No-op when inReplyToURL is empty or not a local Stream.
func (service *Stream) AddReply(session data.Session, inReplyToURL string, replyURL string) error {

	const location = "service.Stream.AddReply"

	// Independent of context ownership (COLLECTIONS-REDESIGN.md Phase 4): a local Stream
	// gets its own Replies collection when replied to, even if the thread's context is remote.

	// RULE: A non-reply (empty inReplyTo) has no parent to attach to.
	if inReplyToURL == "" {
		return nil
	}

	// Resolve the parent. A parent that isn't a local Stream (remote host, or not
	// found) resolves to NotFound, and is not ours to track replies for, so we
	// quietly skip it.
	parent := model.NewStream()

	if err := service.LoadByURL(session, inReplyToURL, &parent); err != nil {

		// If the parent doesn't exist, then there's nothing to link
		if derp.IsNotFound(err) {
			return nil
		}

		// Otherwise it's a legitimate error to return to the caller.
		return derp.Wrap(err, location, "Loading parent Stream", "inReplyTo: "+inReplyToURL)
	}

	// RULE: Only public streams expose reply collections (mirrors CalcContext).
	if !parent.DefaultAllow.IsAnonymous() {
		return nil
	}

	// JIT the parent's Replies collection (concurrency-safe via the unique index).
	collection, err := service.collectionService.LoadOrCreateByStream(session, &parent, model.CollectionTypeReplies)

	if err != nil {
		return derp.Wrap(err, location, "Loading/Creating replies collection", "parentID: "+parent.StreamID.Hex())
	}

	// Add the reply to the collection.
	if err := service.collectionService.AddItem(session, &collection, replyURL, inReplyToURL); err != nil {
		return derp.Wrap(err, location, "Adding reply to collection", "replyURL: "+replyURL)
	}

	// Refresh the parent's denormalized ReplyCount from the collection.
	if err := service.refreshReplyCount(session, &parent, &collection); err != nil {
		return derp.Wrap(err, location, "Refreshing reply count", parent.StreamID.Hex())
	}

	// Station.
	return nil
}

// ReindexReplies re-projects every reply Stream into its parent's Replies collection
// and refreshes counts. It is safe to re-run (idempotent).
func (service *Stream) ReindexReplies(session data.Session) error {

	const location = "service.Stream.ReindexReplies"

	// AddReply is idempotent: AddItem de-dupes by URI and counts are recomputed, so a
	// repeated run converges to the same state.
	replies, err := service.Range(session, exp.NotEqual("inReplyTo", ""))

	if err != nil {
		return derp.Wrap(err, location, "Ranging reply Streams")
	}

	for reply := range replies {
		if err := service.AddReply(session, reply.InReplyTo, reply.ActivityPubURL()); err != nil {
			// Report and continue: one bad row must not abort the whole backfill.
			derp.Report(derp.Wrap(err, location, "Projecting reply", reply.StreamID.Hex()))
		}
	}

	return nil
}

// RemoveReply removes replyURL from the Replies collection of the local Stream that
// inReplyToURL points to, and refreshes the count. No-op when the parent is not a local Stream.
func (service *Stream) RemoveReply(session data.Session, inReplyToURL string, replyURL string) error {

	const location = "service.Stream.RemoveReply"

	if inReplyToURL == "" {
		return nil
	}

	// Resolve the parent. A parent that isn't a local Stream (remote host, or not
	// found) resolves to NotFound, and is not ours to track replies for, so we
	// quietly skip it.
	parent := model.NewStream()

	if err := service.LoadByURL(session, inReplyToURL, &parent); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Loading parent Stream", "inReplyTo: "+inReplyToURL)
	}

	// Locate the parent's Replies collection. If none exists, there is nothing to remove.
	collection := model.NewCollection()

	if err := service.collectionService.LoadByType(session, parent.StreamID, model.CollectionTypeReplies, &collection); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Loading Replies collection", "parentID: "+parent.StreamID.Hex())
	}

	if err := service.collectionService.RemoveItem(session, &collection, replyURL); err != nil {
		return derp.Wrap(err, location, "Removing reply from collection", "replyURL: "+replyURL)
	}

	if err := service.refreshReplyCount(session, &parent, &collection); err != nil {
		return derp.Wrap(err, location, "Refreshing reply count", parent.StreamID.Hex())
	}

	return nil
}

// refreshReplyCount recomputes the parent Stream's denormalized ReplyCount from the live
// Replies collection size and Saves it.
func (service *Stream) refreshReplyCount(session data.Session, parent *model.Stream, collection *model.Collection) error {

	const location = "service.Stream.refreshReplyCount"

	// Recompute-and-Save (not increment) because data.Collection exposes no atomic $inc;
	// a stale overwrite re-derives correctly next time (cf. COLLECTIONS-REDESIGN D4).
	count, err := service.collectionService.CountItems(session, collection)

	if err != nil {
		return derp.Wrap(err, location, "Counting Replies collection", collection.CollectionID.Hex())
	}

	parent.ReplyCount = int(count)

	if err := service.Save(session, parent, "Refreshed reply count"); err != nil {
		return derp.Wrap(err, location, "Saving Stream with refreshed reply count", parent.StreamID.Hex())
	}

	return nil
}

// AddResponseCollectionItem records an inbound remote Like/Dislike/Announce (activityURL) in the
// JIT Likes/Dislikes/Shares collection of the local Stream that targetURL points to, and refreshes
// the matching count. This is the inbound-federation counterpart to the Response service's funnel,
// which only handles LOCAL responses (in-app, Mastodon, intent). No-op when activityType is not a
// projected response type, or targetURL is empty / not a local public Stream.
func (service *Stream) AddResponseCollectionItem(session data.Session, targetURL string, activityType string, activityURL string) error {

	const location = "service.Stream.AddResponseCollectionItem"

	// RULE: Without a target or an activity URL there is nothing to project.
	if targetURL == "" || activityURL == "" {
		return nil
	}

	// Resolve the target. A target that isn't a local Stream (remote host, or not found)
	// resolves to NotFound, and is not ours to track responses for, so we quietly skip it.
	target := model.NewStream()

	if err := service.LoadByURL(session, targetURL, &target); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading target Stream", "target: "+targetURL)
	}

	// RULE: Only public streams expose response collections (mirrors AddReply). This object-side
	// policy is the reason inbound projection resolves the Stream here rather than in the primitive.
	if !target.DefaultAllow.IsAnonymous() {
		return nil
	}

	// Project via the shared primitive, keyed by the activity's own (resolvable) URL.
	collection, collectionType, err := service.collectionService.ProjectResponse(session, &target, activityType, activityURL, true)

	if err != nil {
		return derp.Wrap(err, location, "Projecting response into collection", "activityURL: "+activityURL)
	}

	if collectionType == "" {
		return nil
	}

	// Refresh the target's denormalized count for this response type.
	if err := service.refreshResponseCount(session, &target, &collection, collectionType); err != nil {
		return derp.Wrap(err, location, "Refreshing response count", target.StreamID.Hex())
	}

	return nil
}

// RemoveResponseCollectionItem removes an inbound remote Like/Dislike/Announce (activityURL) from the
// Likes/Dislikes/Shares collection of the local Stream that targetURL points to, and refreshes the
// matching count. It is the symmetric counterpart to AddResponseCollectionItem, called when an Undo
// (or Delete) of the original activity arrives. No-op when the target is not a local Stream, or no
// such collection/item exists.
func (service *Stream) RemoveResponseCollectionItem(session data.Session, targetURL string, activityType string, activityURL string) error {

	const location = "service.Stream.RemoveResponseCollectionItem"

	// RULE: Without a target or an activity URL there is nothing to remove.
	if targetURL == "" || activityURL == "" {
		return nil
	}

	// Resolve the target. A target that isn't a local Stream (remote host, or not found)
	// resolves to NotFound, and is not ours to track responses for, so we quietly skip it.
	target := model.NewStream()

	if err := service.LoadByURL(session, targetURL, &target); err != nil {
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Loading target Stream", "target: "+targetURL)
	}

	// Project (removal) via the shared primitive. A zero collectionType means there was nothing
	// to remove (non-projected type, or the Stream has no such collection yet).
	collection, collectionType, err := service.collectionService.ProjectResponse(session, &target, activityType, activityURL, false)

	if err != nil {
		return derp.Wrap(err, location, "Removing response from collection", "activityURL: "+activityURL)
	}

	if collectionType == "" {
		return nil
	}

	if err := service.refreshResponseCount(session, &target, &collection, collectionType); err != nil {
		return derp.Wrap(err, location, "Refreshing response count", target.StreamID.Hex())
	}

	return nil
}

// refreshResponseCount recomputes the target Stream's denormalized Like/Dislike/Share count from the
// live collection size and Saves it. Mirrors refreshReplyCount, but selects the count field by type.
func (service *Stream) refreshResponseCount(session data.Session, target *model.Stream, collection *model.Collection, collectionType string) error {

	const location = "service.Stream.refreshResponseCount"

	// Recompute-and-Save (not increment) because data.Collection exposes no atomic $inc;
	// a stale overwrite re-derives correctly next time (cf. COLLECTIONS-REDESIGN D4).
	count, err := service.collectionService.CountItems(session, collection)

	if err != nil {
		return derp.Wrap(err, location, "Counting response collection", collection.CollectionID.Hex())
	}

	switch collectionType {

	case model.CollectionTypeLikes:
		target.LikeCount = int(count)

	case model.CollectionTypeDislikes:
		target.DislikeCount = int(count)

	case model.CollectionTypeShares:
		target.ShareCount = int(count)

	default:
		return derp.Internal(location, "Unexpected collection type", collectionType)
	}

	if err := service.Save(session, target, "Refreshed response count"); err != nil {
		return derp.Wrap(err, location, "Saving Stream with refreshed response count", target.StreamID.Hex())
	}

	return nil
}
