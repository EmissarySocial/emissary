package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
)

const filterMaxRounds = 4 // Most batches read while looking for enough posts that pass a filter

// statusFilter returns a test for the profile and timeline filters clients ask for.
// A reply to the owner's own post stays in, so that a thread isn't split from its first post.
func statusFilter(excludeReplies bool, onlyMedia bool, ownerID string) func(object.Status) bool {

	return func(status object.Status) bool {

		if onlyMedia && len(status.MediaAttachments) == 0 {
			return false
		}

		if excludeReplies && status.InReplyToID != "" && status.InReplyToAccountID != ownerID {
			return false
		}

		return true
	}
}

// collectFiltered reads batches of posts, newest first, until it has limit that pass keep or runs out.
// fetch returns one batch created before the given time (0 means no bound); kept posts come back with their statuses.
func collectFiltered(limit int, fetch func(before int64) ([]model.Stream, error), convert func([]model.Stream) []object.Status, keep func(object.Status) bool) ([]model.Stream, []object.Status, error) {

	streams := []model.Stream{}
	statuses := []object.Status{}
	before := int64(0)

	for round := 0; round < filterMaxRounds && len(statuses) < limit; round++ {

		// Read the next batch of older posts
		batch, err := fetch(before)

		if err != nil {
			return nil, nil, err
		}

		if len(batch) == 0 {
			break
		}

		// Keep the posts that pass, until there are enough
		converted := convert(batch)

		for index := range batch {

			if len(statuses) >= limit {
				break
			}

			if keep(converted[index]) {
				streams = append(streams, batch[index])
				statuses = append(statuses, converted[index])
			}
		}

		// A short batch means there is nothing older to read
		if len(batch) < limit {
			break
		}

		before = batch[len(batch)-1].CreateDate
	}

	return streams, statuses, nil
}
