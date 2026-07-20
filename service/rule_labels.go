package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LabelDocuments stamps each document's per-viewer rule verdict into its Metadata.Labels,
// using ONE indexed rules query per chunk of documents (the union of every document's match
// keys; the engine re-checks membership per document, so over-fetching is safe).
func (service *Rule) LabelDocuments(session data.Session, userID primitive.ObjectID, documents []streams.Document) {

	const location = "service.Rule.LabelDocuments"

	now := time.Now().Unix()

	for chunkStart := 0; chunkStart < len(documents); chunkStart += labelChunkSize {

		chunk := documents[chunkStart:min(chunkStart+labelChunkSize, len(documents))]

		// Collect each document's match keys: the same set the asrules checker evaluates,
		// so list views and single-document loads can never disagree about a verdict.
		perDocument := make([][]string, len(chunk))
		keys := make([]string, 0, len(chunk)*4)

		for index := range chunk {
			perDocument[index] = append(model.ActorMatchKeys(chunk[index].ID()), model.DocumentMatchKeys(chunk[index])...)
			keys = append(keys, perDocument[index]...)
		}

		// Query the viewer's Rules for the whole chunk at once
		rules, err := service.QueryByMatchKeys(session, userID, keys)

		// RULE: display fails OPEN. Refusal-at-fetch is asrules' job; this stamp only drives
		// placeholders and label chips, so a database blip serves unlabeled documents instead
		// of breaking the page.
		if err != nil {
			derp.Report(derp.Wrap(err, location, "Querying rules for document labels; serving unlabeled"))
			return
		}

		// Stamp each document's verdict into its per-viewer Metadata
		for index := range chunk {
			disposition := model.NewRuleDispositionForKeys(perDocument[index], rules, now)
			chunk[index].Metadata.Labels = disposition.LabelSet()
		}
	}
}

// LabelNotifications stamps each Notification's per-viewer rule verdict into its transient
// Labels field, using ONE indexed rules query per chunk. The verdict is derived from the
// notification's snapshotted Actor (R8: derive at render, never record).
func (service *Rule) LabelNotifications(session data.Session, userID primitive.ObjectID, notifications []model.Notification) {

	const location = "service.Rule.LabelNotifications"

	now := time.Now().Unix()

	for chunkStart := 0; chunkStart < len(notifications); chunkStart += labelChunkSize {

		chunk := notifications[chunkStart:min(chunkStart+labelChunkSize, len(notifications))]

		// Collect each notification's match keys from its snapshotted actor
		perNotification := make([][]string, len(chunk))
		keys := make([]string, 0, len(chunk)*4)

		for index := range chunk {
			perNotification[index] = model.ActorMatchKeys(chunk[index].Actor.ProfileURL)
			keys = append(keys, perNotification[index]...)
		}

		// Query the viewer's Rules for the whole chunk at once
		rules, err := service.QueryByMatchKeys(session, userID, keys)

		// RULE: display fails OPEN (same posture as LabelDocuments)
		if err != nil {
			derp.Report(derp.Wrap(err, location, "Querying rules for notification labels; serving unlabeled"))
			return
		}

		// Stamp each notification's verdict into its transient Labels field
		for index := range chunk {
			disposition := model.NewRuleDispositionForKeys(perNotification[index], rules, now)
			chunk[index].Labels = disposition.LabelSet()
		}
	}
}

// LabelSearchResults stamps each SearchResult's per-viewer rule verdict into its transient
// Labels field, using ONE indexed rules query per chunk. A SearchResult is a local index row,
// not a cached ActivityStream document, so its match keys come from its URL and its tags.
func (service *Rule) LabelSearchResults(session data.Session, userID primitive.ObjectID, results []model.SearchResult) {

	const location = "service.Rule.LabelSearchResults"

	now := time.Now().Unix()

	for chunkStart := 0; chunkStart < len(results); chunkStart += labelChunkSize {

		chunk := results[chunkStart:min(chunkStart+labelChunkSize, len(results))]

		// Collect each result's match keys: its URL (ACTOR + DOMAIN keys) and its tags (TAG keys)
		perResult := make([][]string, len(chunk))
		keys := make([]string, 0, len(chunk)*4)

		for index := range chunk {

			resultKeys := model.ActorMatchKeys(chunk[index].URL)

			for _, tag := range chunk[index].Tags {
				if tagKey := model.RuleMatchKey(model.RuleTypeTag, tag); tagKey != "" {
					resultKeys = append(resultKeys, tagKey)
				}
			}

			perResult[index] = resultKeys
			keys = append(keys, resultKeys...)
		}

		// Query the viewer's Rules for the whole chunk at once
		rules, err := service.QueryByMatchKeys(session, userID, keys)

		// RULE: display fails OPEN (same posture as LabelDocuments)
		if err != nil {
			derp.Report(derp.Wrap(err, location, "Querying rules for search labels; serving unlabeled"))
			return
		}

		// Stamp each result's verdict into its transient Labels field
		for index := range chunk {
			disposition := model.NewRuleDispositionForKeys(perResult[index], rules, now)
			chunk[index].Labels = disposition.LabelSet()
		}
	}
}
