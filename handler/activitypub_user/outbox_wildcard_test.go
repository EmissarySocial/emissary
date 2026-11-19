package activitypub_user

import (
	"encoding/json"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// unaddressedActivity returns an activity of the provided type, carrying the provided properties
// and no addressing beyond them
func unaddressedActivity(activityType string, properties mapof.Any) streams.Document {

	value := mapof.Any{
		vocab.PropertyType:   activityType,
		vocab.PropertyActor:  "https://this.example/@alice",
		vocab.PropertyObject: "https://remote.example/notes/1",
	}

	for name, property := range properties {
		value[name] = property
	}

	return streams.NewDocument(value)
}

// TestOutboxWildcard_RefusesUnaddressedActivity verifies that an activity naming no recipient is
// refused as a bad request, before anything is stored or written to the response
func TestOutboxWildcard_RefusesUnaddressedActivity(t *testing.T) {

	// O2: a Mastodon-shaped Like, Follow, Undo, Accept, Block or Move carries no `to`, and was stored
	// and delivered to no one. The Context has no factory, so storing it would nil-panic the test.
	activityTypes := []string{
		vocab.ActivityTypeLike,
		vocab.ActivityTypeFollow,
		vocab.ActivityTypeUndo,
		vocab.ActivityTypeAccept,
		vocab.ActivityTypeReject,
		vocab.ActivityTypeBlock,
		vocab.ActivityTypeMove,
		vocab.ActivityTypeAnnounce,
	}

	for _, activityType := range activityTypes {

		context, recorder := newResponseContext()
		err := outbox_Wildcard(context, unaddressedActivity(activityType, nil))

		require.True(t, derp.IsBadRequest(err), "%s: want BadRequest, got %v", activityType, err)
		require.False(t, context.context.Response().Committed, "%s: nothing is written for a refusal", activityType)
		require.Empty(t, recorder.Header().Get("X-Activity-Id"), "%s: no activity was stored", activityType)
	}
}

// TestRequireRecipients verifies which addressing names a recipient that delivery can reach
func TestRequireRecipients(t *testing.T) {

	// test checks whether one set of properties is accepted
	test := func(name string, properties mapof.Any, accepted bool) {
		t.Run(name, func(t *testing.T) {

			err := requireRecipients(unaddressedActivity(vocab.ActivityTypeLike, properties))

			if accepted {
				require.NoError(t, err)
				return
			}

			require.True(t, derp.IsBadRequest(err), "want BadRequest, got %v", err)
		})
	}

	// Each addressing property counts on its own
	test("to", mapof.Any{vocab.PropertyTo: "https://remote.example/@bob"}, true)
	test("cc", mapof.Any{vocab.PropertyCC: []string{"https://remote.example/@bob"}}, true)
	test("bto", mapof.Any{vocab.PropertyBTo: []any{"https://remote.example/@bob"}}, true)
	test("bcc", mapof.Any{vocab.PropertyBCC: "https://remote.example/@bob"}, true)
	test("Public alone", mapof.Any{vocab.PropertyTo: vocab.NamespaceActivityStreamsPublic}, true)
	test("recipient as an object", mapof.Any{vocab.PropertyTo: []any{mapof.Any{vocab.PropertyID: "https://remote.example/@bob"}}}, true)
	test("one real recipient among empty ones", mapof.Any{vocab.PropertyTo: []any{"", nil}, vocab.PropertyCC: "https://remote.example/@bob"}, true)

	// Addressing that names no one is refused
	test("no addressing", nil, false)
	test("empty string", mapof.Any{vocab.PropertyTo: ""}, false)
	test("empty list", mapof.Any{vocab.PropertyTo: []string{}, vocab.PropertyCC: []any{}}, false)
	test("list of empty strings", mapof.Any{vocab.PropertyTo: []string{"", ""}}, false)
	test("null", mapof.Any{vocab.PropertyTo: nil}, false)
	test("object with no id", mapof.Any{vocab.PropertyTo: []any{mapof.Any{vocab.PropertyType: "Person"}}}, false)

	// RULE: Delivery reads only to, cc, bto and bcc, so audience and a mention name no one
	test("audience alone", mapof.Any{vocab.PropertyAudience: "https://remote.example/@bob"}, false)
	test("mention alone", mapof.Any{vocab.PropertyTag: []any{mapof.Any{vocab.PropertyType: vocab.LinkTypeMention, vocab.PropertyHref: "https://remote.example/@bob"}}}, false)
}

// FuzzRequireRecipients verifies that any JSON activity is answered without panicking, and that it
// is refused exactly when it names no recipient
func FuzzRequireRecipients(f *testing.F) {

	f.Add(`{"type":"Like","to":"https://remote.example/@bob"}`)
	f.Add(`{"type":"Follow","object":"https://remote.example/@bob"}`)
	f.Add(`{"to":[{"type":"Person"},null,1,true,[["x"]]],"bcc":{"id":{"id":"y"}}}`)
	f.Add(`{"to":"","cc":[""]}`)
	f.Add(`null`)

	f.Fuzz(func(t *testing.T, document string) {

		value := mapof.NewAny()

		if err := json.Unmarshal([]byte(document), &value); err != nil {
			return
		}

		activity := streams.NewDocument(value)
		err := requireRecipients(activity)

		// RULE: Error XOR a recipient
		if len(activity.Recipients()) == 0 {
			require.True(t, derp.IsBadRequest(err), "an activity naming no one must be refused")
			return
		}

		require.NoError(t, err)
	})
}
