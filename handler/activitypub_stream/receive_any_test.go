package activitypub_stream

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestRequireOwnObject confirms that a Create or Update may carry only an object on its actor's own
// origin, and that other activity types are not checked here.
func TestRequireOwnObject(t *testing.T) {

	const actor = "https://good.example/@alice"

	test := func(name string, activityType string, object any, allowed bool) {
		t.Run(name, func(t *testing.T) {

			activity := streams.NewDocument(mapof.Any{
				vocab.PropertyType:   activityType,
				vocab.PropertyActor:  actor,
				vocab.PropertyObject: object,
			})

			err := requireOwnObject(activity)

			if allowed {
				require.NoError(t, err)
				return
			}

			require.True(t, derp.IsForbidden(err), "want Forbidden, got %v", err)
		})
	}

	own := mapof.Any{vocab.PropertyID: "https://good.example/notes/1", vocab.PropertyType: vocab.ObjectTypeNote}
	foreign := mapof.Any{vocab.PropertyID: "https://victim.example/@bob", vocab.PropertyType: vocab.ActorTypePerson}
	uuid := mapof.Any{vocab.PropertyID: "urn:uuid:550e8400-e29b-41d4-a716-446655440000", vocab.PropertyType: vocab.ObjectTypeNote}
	anonymous := mapof.Any{vocab.PropertyType: vocab.ObjectTypeNote}

	test("create own object", vocab.ActivityTypeCreate, own, true)
	test("update own object", vocab.ActivityTypeUpdate, own, true)
	test("create own link", vocab.ActivityTypeCreate, "https://good.example/notes/1", true)
	test("create foreign object", vocab.ActivityTypeCreate, foreign, false)
	test("update foreign object", vocab.ActivityTypeUpdate, foreign, false)
	test("create foreign link", vocab.ActivityTypeCreate, "https://victim.example/@bob", false)
	test("create urn:uuid object", vocab.ActivityTypeCreate, uuid, false)
	test("create object with no id", vocab.ActivityTypeCreate, anonymous, false)
	test("announce foreign object", vocab.ActivityTypeAnnounce, foreign, true)
	test("like foreign object", vocab.ActivityTypeLike, foreign, true)
}

// TestBoostAny_RejectsForeignObjectBeforeAnyLookup confirms that a Stream refuses a Create whose object
// claims another origin, and does so before it reaches the factory.
func TestBoostAny_RejectsForeignObjectBeforeAnyLookup(t *testing.T) {

	// BUG-223: the object would otherwise be saved to the shared cache under the victim's id, key and
	// all.  The Context has no factory, so any lookup before the check would nil-panic the test.
	context := Context{actor: &model.StreamActor{BoostInbox: true}}

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyID:    "https://mallory.example/activities/1",
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: "https://mallory.example/@mallory",
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyID:   "https://victim.example/@bob",
			vocab.PropertyType: vocab.ActorTypePerson,
			"publicKey":        mapof.Any{"publicKeyPem": "PEM-ATTACKER"},
		},
	})

	err := BoostAny(context, activity)

	require.True(t, derp.IsForbidden(err), "want Forbidden, got %v", err)
}
