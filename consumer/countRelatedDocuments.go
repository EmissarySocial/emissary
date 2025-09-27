package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// CountRelatedDocuments is run after an ActivityStream document is cached,
// and searches the ActivityStream cache for "related" documents: InReplyTo, Likes, and Shares.
func CountRelatedDocuments(factory *service.Factory, args mapof.Any) queue.Result {

	// Temporarily disabling this worker to see if it's the reason mongo CPU usage is so high
	return queue.Success()

	/*
		const location = "consumer.CountRelatedDocuments"

		// Collect arguments
		url := args.GetString("url")
		actorType := args.GetString("actorType")
		actorToken := args.GetString("actorID")
		actorID, err := primitive.ObjectIDFromHex(actorToken)

		if err != nil {
			return queue.Failure(derp.Wrap(err, location, "Invalid actorID"))
		}

		// Get ActivityStreams service
		activityService := factory.ActivityStream(actorType, actorID)
		cacheClient := activityService.CacheClient()

		// Create a database transaction
		ctx, cancel := timeoutContext(20)
		defer cancel()

		_, err = factory.CommonDatabase().WithTransaction(ctx, func(session data.Session) (any, error) {

			// Load the target document
			document, err := cacheClient.Load(url)

			if err != nil {
				return nil, derp.Wrap(err, location, "Unable to load document", url)
			}

			documentID := document.ID()
			changed := false

			// Count Replies
			replyCount, err := cacheClient.CountRelatedValues(session, vocab.RelationTypeReply, documentID)
			changed = document.Metadata.SetRelationCount(vocab.RelationTypeReply, replyCount) || changed

			if err != nil {
				return nil, derp.Wrap(err, location, "Unable to count replies", documentID)
			}

			// Count Announces
			announceCount, err := cacheClient.CountRelatedValues(session, vocab.RelationTypeAnnounce, documentID)
			changed = document.Metadata.SetRelationCount(vocab.RelationTypeAnnounce, announceCount) || changed

			if err != nil {
				return nil, derp.Wrap(err, location, "Unable to count announces", documentID)
			}

			// Count Likes
			likeCount, err := cacheClient.CountRelatedValues(session, vocab.RelationTypeLike, documentID)
			changed = document.Metadata.SetRelationCount(vocab.RelationTypeLike, likeCount) || changed

			if err != nil {
				return nil, derp.Wrap(err, location, "Unable to count likes", documentID)
			}

			// If values have changed, then update the database now
			if changed {
				if err := cacheClient.Save(document); err != nil {
					return nil, derp.Wrap(err, location, "Unable to save document", documentID)
				}
			}

			// Twice the `nil` => twice the success!
			return nil, nil
		})

		// Wrap transaction errors as a Queue.Error
		if err != nil {
			return queue.Error(err)
		}

		// Only one success this time, but still it's good.
		return queue.Success()
	*/
}
