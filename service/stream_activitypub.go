package service

import (
	"iter"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * ActivityPub API
 ******************************************/

// JSONLDGetter returns a new JSONLDGetter for the provided stream
func (service *Stream) JSONLDGetter(stream *model.Stream) StreamJSONLDGetter {
	return NewStreamJSONLDGetter(service, stream)
}

// GetJSONLD returns a map document that conforms to the ActivityStreams 2.0 spec.
// This map will still need to be marshalled into JSON
func (service *Stream) JSONLD(stream *model.Stream) mapof.Any {
	result := mapof.Any{
		vocab.AtContext:         sliceof.Any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyID:        stream.ActivityPubURL(),
		vocab.PropertyType:      stream.SocialRole,
		vocab.PropertyURL:       stream.URL,
		vocab.PropertyPublished: time.Unix(stream.PublishDate, 0).UTC().Format(time.RFC3339),
	}

	if stream.Label != "" {
		result[vocab.PropertyName] = stream.Label
	}

	/* REMOVED because this is used by Mastodon as a Content Warning
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

	if len(stream.Hashtags) > 0 {
		result[vocab.PropertyTag] = slice.Map(stream.Hashtags, service.HashtagAsJSONLD)
	}

	// NOTE: According to Mastodon ActivityPub guide (https://docs.joinmastodon.org/spec/activitypub/)
	// putting as:public in the To field means that this mesage is public, and "listed"
	// putting as:public in the Cc field means that this message is public, but "unlisted"
	// and leaving as:public out entirely means that this message is "private" -- for whatever that's worth...

	if stream.DefaultAllowAnonymous() {
		result[vocab.PropertyTo] = []string{vocab.NamespaceActivityStreamsPublic}
	}

	// Custom behaviors for different stream types
	switch stream.SocialRole {

	case vocab.ObjectTypeAudio:

		// Size (in bytes)
		// Bitrate
		// Duration
		// Library (custom Funkwhale type)

		if attachments, err := service.attachmentService.QueryByCategory(model.AttachmentObjectTypeStream, stream.StreamID, vocab.ObjectTypeAudio); err == nil {
			link := make([]mapof.Any, 0, len(attachments))

			for _, attachment := range attachments {
				link = append(link, mapof.Any{
					vocab.PropertyType:      vocab.CoreTypeLink,
					vocab.PropertyHref:      stream.ActivityPubURL() + "/attachments/" + attachment.AttachmentID.Hex() + ".mpg",
					vocab.PropertyMediaType: "audio/mpeg",
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
	if attachments, err := service.attachmentService.QueryByObjectID(model.AttachmentObjectTypeStream, stream.StreamID); err == nil {

		attachmentJSON := make([]mapof.Any, 0, len(attachments))
		for _, attachment := range attachments {
			attachmentJSON = append(attachmentJSON, attachment.JSONLD())
		}

		result[vocab.PropertyAttachment] = attachmentJSON
	}

	// Try to apply the "social mapping" to the stream
	schma := service.activityStreamSchema()
	if template, err := service.templateService.Load(stream.TemplateID); err == nil {
		result[vocab.PropertyType] = template.SocialRole
		if template.SocialRules.NotEmpty() {
			err := template.SocialRules.Execute(schma, stream, schma, &result)
			derp.Report(err)
		}
	}

	return result
}

// HashtagAsJSONLD returns a JSON-LD map document that represents a hashtag
func (service *Stream) HashtagAsJSONLD(tag string) mapof.String {
	return mapof.String{
		vocab.PropertyType: vocab.LinkTypeHashtag,
		vocab.PropertyName: tag,
		vocab.PropertyHref: service.host + "/search?q=%23=" + tag,
	}
}

func (service *Stream) ActivityPubURL(streamID primitive.ObjectID) string {
	return service.host + "/" + streamID.Hex()
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided Stream.
func (service *Stream) ActivityPubActor(streamID primitive.ObjectID) (outbox.Actor, error) {

	const location = "service.Stream.ActivityPubActor"

	// Try to load the user's keys from the database
	encryptionKey := model.NewEncryptionKey()
	if err := service.keyService.LoadByParentID(model.EncryptionKeyTypeStream, streamID, &encryptionKey); err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error loading encryption key", streamID)
	}

	// Extract the Private Key from the Encryption Key
	privateKey, err := service.keyService.GetPrivateKey(&encryptionKey)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error extracting private key", encryptionKey)
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActivityPubURL(streamID),
		privateKey,
		outbox.WithFollowers(service.RangeActivityPubFollowers(streamID)),
		outbox.WithClient(service.activityStream),
		// TODO: Restore Queue:: , outbox.WithQueue(service.queue))
	)

	return actor, nil
}

// ActivityPubActor returns an ActivityPub Actor object ** WHICH INCLUDES ENCRYPTION KEYS **
// for the provided User.
func (service *Stream) RangeActivityPubFollowers(streamID primitive.ObjectID) iter.Seq[string] {

	return func(yield func(string) bool) {

		// Retrieve all Followers for this Stream
		followers := service.followerService.RangeActivityPubByType(model.FollowerTypeStream, streamID)

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
