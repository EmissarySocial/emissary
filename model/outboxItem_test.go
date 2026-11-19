package model

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestOutboxItemSchema returns the rosetta schema that describes a TestOutboxItem
func TestOutboxItemSchema(t *testing.T) {

	annotation := NewOutboxItem()
	s := schema.New(OutboxItemSchema())

	table := []tableTestItem{
		{"activityId", "123456781234567812345678", nil},
		{"actorType", "User", nil},
		{"actorId", "876543218765432187654321", nil},
		{"recipients.0", "Public", nil},
		{"activity.to", "Public", nil},
		{"activity.id", "http://example.com/activities/1", nil},
		{"activity.type", "Note", nil},
		{"activity.published", "2024-01-01T12:00:00Z", nil},
		{"activity.attributedTo", "http://example.com/users/alice", nil},
		{"activity.content", "<p>Hello, world!</p>", nil},
		{"url", "https://example.com/outbox/1", nil},
		{"activityType", "Create", nil},
		{"objectId", "https://example.com/notes/1", nil},
		{"permissions.0", "086753090867530908675309", nil},
		{"permissions.1", "086753090867530908675309", nil},
	}

	tableTest_Schema(t, &s, &annotation, table)
}

// TestOutboxItemSchema_ArrayLimits verifies that the lists refuse an index at their MaxLength
func TestOutboxItemSchema_ArrayLimits(t *testing.T) {

	// BUG-244: an unbounded index made the server allocate a list of any length
	s := schema.New(OutboxItemSchema())

	for path, limit := range map[string]int{"recipients": 1024, "permissions": 256} {
		item := NewOutboxItem()
		require.NoError(t, s.Set(&item, path+"."+strconv.Itoa(limit-1), "086753090867530908675309"), path)
		require.Error(t, s.Set(&item, path+"."+strconv.Itoa(limit), "086753090867530908675309"), path)
		require.Error(t, s.Set(&item, path+".5000000", "086753090867530908675309"), path)
	}
}

// TestNewOutboxItem_PermissionsAreEmpty verifies that a new item is readable by no one until permissions are set
func TestNewOutboxItem_PermissionsAreEmpty(t *testing.T) {
	item := NewOutboxItem()
	require.NotNil(t, item.Permissions)
	require.Empty(t, item.Permissions)
	require.False(t, item.Permissions.IsAnonymous())
}

// TestOutboxItem_GetJSONLD verifies that blind recipients are removed at every level, from a copy
func TestOutboxItem_GetJSONLD(t *testing.T) {

	// BUG-07: bto and bcc must never be disclosed to another party
	item := NewOutboxItem()
	item.Activity = mapof.Any{
		vocab.PropertyID:   "https://example.com/activities/1",
		vocab.PropertyType: vocab.ActivityTypeCreate,
		vocab.PropertyTo:   []any{"https://example.com/users/visible"},
		vocab.PropertyBTo:  []any{"https://example.com/users/secret"},
		vocab.PropertyBCC:  "https://example.com/users/secret",
		vocab.PropertyObject: map[string]any{
			vocab.PropertyType: vocab.ObjectTypeNote,
			vocab.PropertyBCC:  "https://example.com/users/deeperSecret",
		},
	}

	result := item.GetJSONLD()

	// The result carries everything but the blind fields...
	require.Equal(t, "https://example.com/activities/1", result[vocab.PropertyID])
	require.Equal(t, []any{"https://example.com/users/visible"}, result[vocab.PropertyTo])
	require.NotContains(t, result, vocab.PropertyBTo)
	require.NotContains(t, result, vocab.PropertyBCC)
	require.NotContains(t, result[vocab.PropertyObject], vocab.PropertyBCC)

	// ...and the stored activity keeps them, because delivery reads them
	require.Contains(t, item.Activity, vocab.PropertyBTo)
	require.Contains(t, item.Activity, vocab.PropertyBCC)
	require.Contains(t, item.Activity[vocab.PropertyObject], vocab.PropertyBCC)
}

// TestOutboxItem_GetJSONLD_DatabaseTypes verifies that an activity read from MongoDB is stripped
// inside the driver's own document and array types, which keep their types
func TestOutboxItem_GetJSONLD_DatabaseTypes(t *testing.T) {

	objectID := primitive.NewObjectID()

	item := NewOutboxItem()
	item.Activity = mapof.Any{
		"_id":       objectID,
		"published": primitive.DateTime(1700000000000),
		"bcc":       primitive.A{"https://example.com/@carol"},
		"object": primitive.M{
			"bto":        primitive.A{"https://example.com/@bob"},
			"attachment": primitive.A{primitive.M{"bcc": "https://example.com/@dave", "name": "a"}},
		},
	}

	require.Equal(t, mapof.Any{
		"_id":       objectID,
		"published": primitive.DateTime(1700000000000),
		"object": primitive.M{
			"attachment": primitive.A{primitive.M{"name": "a"}},
		},
	}, item.GetJSONLD())

	require.Contains(t, item.Activity, "bcc")
	require.Contains(t, item.Activity["object"], "bto")
}

// TestOutboxItem_GetJSONLD_CannotBeStripped verifies that an activity holding a value that could
// hide a blind recipient is served as an empty document
func TestOutboxItem_GetJSONLD_CannotBeStripped(t *testing.T) {

	item := NewOutboxItem()
	item.Activity = mapof.Any{
		"type":   "Create",
		"object": primitive.D{{Key: "bcc", Value: "https://example.com/@blind"}},
	}

	require.Empty(t, item.GetJSONLD())
}

// TestOutboxItem_GetJSONLD_Empty verifies that an item with no activity returns an empty document
func TestOutboxItem_GetJSONLD_Empty(t *testing.T) {
	require.Empty(t, NewOutboxItem().GetJSONLD())
	require.Empty(t, OutboxItem{}.GetJSONLD())
}

// TestOutboxItem_Created verifies that Created returns the journal's CreateDate
func TestOutboxItem_Created(t *testing.T) {
	item := OutboxItem{Journal: journal.Journal{CreateDate: 1700000000123}}
	require.Equal(t, int64(1700000000123), item.Created())
	require.Zero(t, NewOutboxItem().Created())
}

// TestOutboxItem_IsVisibleTo verifies who may read a single item from an outbox
func TestOutboxItem_IsVisibleTo(t *testing.T) {

	const author = "https://example.com/@author"
	const recipient = "https://remote.example/@recipient"
	const stranger = "https://remote.example/@stranger"

	circleID := primitive.NewObjectID()
	anonymous := NewAnonymousPermissions()

	// newItem returns a direct message from the author to the recipient, readable by the provided permissions
	newItem := func(permissions Permissions) OutboxItem {
		item := NewOutboxItem()
		item.Permissions = permissions
		item.Activity = mapof.Any{
			vocab.PropertyActor: author,
			vocab.PropertyTo:    []string{recipient, author + "/pub/followers"},
			vocab.PropertyBCC:   []string{"https://remote.example/@blind"},
		}
		item.CalcRecipients()
		return item
	}

	private := newItem(NewPermissions())
	require.False(t, private.IsVisibleTo(anonymous, ""), "an unsigned viewer cannot read a private item")
	require.False(t, private.IsVisibleTo(anonymous, stranger), "a signed stranger cannot read a private item")
	require.False(t, private.IsVisibleTo(nil, ""), "a viewer with no permissions reads nothing")
	require.True(t, private.IsVisibleTo(anonymous, recipient), "a named recipient can read it")
	require.True(t, private.IsVisibleTo(anonymous, "https://remote.example/@blind"), "a blind recipient can read it")
	require.True(t, private.IsVisibleTo(anonymous, author), "its own actor can read it")
	require.False(t, private.IsVisibleTo(anonymous, author+"/pub/followers/x"), "a collection it names admits no one by itself")

	public := newItem(NewAnonymousPermissions())
	require.True(t, public.IsVisibleTo(anonymous, ""), "anyone can read a public item")

	circle := newItem(Permissions{circleID})
	require.False(t, circle.IsVisibleTo(anonymous, stranger), "a stranger is outside the circle")
	require.True(t, circle.IsVisibleTo(Permissions{MagicGroupIDAnonymous, circleID}, stranger), "a circle member can read it")

	// An item with no actor and no recipients never matches an empty viewer
	empty := NewOutboxItem()
	require.False(t, empty.IsVisibleTo(anonymous, ""))
}

// TestOutboxItem_IsVisibleTo_EmbeddedActor verifies that an activity which embeds its actor as an
// object still admits its publisher, and nobody else by that route
func TestOutboxItem_IsVisibleTo_EmbeddedActor(t *testing.T) {

	// O8: the actor was read with GetString, which returns "" for an object, so the publisher of
	// a private activity that embedded its actor could not read it back
	const author = "https://example.com/@author"

	item := NewOutboxItem()
	item.Activity = mapof.Any{
		vocab.PropertyActor: mapof.Any{vocab.PropertyID: author, vocab.PropertyType: vocab.ActorTypePerson},
		vocab.PropertyTo:    []string{"https://remote.example/@recipient"},
	}
	item.CalcRecipients()

	require.True(t, item.IsVisibleTo(NewAnonymousPermissions(), author))
	require.False(t, item.IsVisibleTo(NewAnonymousPermissions(), "https://remote.example/@stranger"))

	// An embedded actor with no id names no one, so it admits no one
	item.Activity[vocab.PropertyActor] = mapof.Any{vocab.PropertyType: vocab.ActorTypePerson}
	require.False(t, item.IsVisibleTo(NewAnonymousPermissions(), author))
}

// TestOutboxItem_CalcRecipients verifies that every addressing property is read, that a recipient
// written as an object counts by its id, and that nothing empty or repeated is stored
func TestOutboxItem_CalcRecipients(t *testing.T) {

	// test checks the recipients calculated for one activity
	test := func(name string, activity mapof.Any, expected ...string) {
		t.Run(name, func(t *testing.T) {
			item := NewOutboxItem()
			item.Activity = activity
			item.CalcRecipients()
			require.ElementsMatch(t, expected, []string(item.Recipients))
		})
	}

	test("strings in every property",
		mapof.Any{"to": []any{"https://a.example/1"}, "cc": "https://a.example/2", "bto": []string{"https://a.example/3"}, "bcc": []any{"https://a.example/4"}},
		"https://a.example/1", "https://a.example/2", "https://a.example/3", "https://a.example/4")

	// O3: a recipient written as an object was stored as an empty string, so it could not read
	// the activity back
	test("objects count by their id",
		mapof.Any{"to": []any{mapof.Any{"id": "https://a.example/1", "type": "Person"}, "https://a.example/2"}, "cc": map[string]any{"id": "https://a.example/3"}},
		"https://a.example/1", "https://a.example/2", "https://a.example/3")

	test("an object with no id names no one",
		mapof.Any{"to": []any{mapof.Any{"type": "Person"}, "https://a.example/1"}},
		"https://a.example/1")

	test("repeats are stored once",
		mapof.Any{"to": []any{"https://a.example/1", mapof.Any{"id": "https://a.example/1"}}, "bcc": "https://a.example/1"},
		"https://a.example/1")

	test("an activity with no addressing has no recipients", mapof.Any{"type": "Create"})
	test("a nil activity has no recipients", nil)
	test("other properties are not addressing", mapof.Any{"audience": "https://a.example/1", "actor": "https://a.example/2"})
}

// TestOutboxItem_CalcRecipients_ObjectRecipientReads verifies that a recipient written as an object
// can read the activity back
func TestOutboxItem_CalcRecipients_ObjectRecipientReads(t *testing.T) {

	item := NewOutboxItem()
	item.Activity = mapof.Any{
		vocab.PropertyActor: "https://example.com/@author",
		vocab.PropertyTo:    []any{mapof.Any{vocab.PropertyID: "https://remote.example/@recipient"}},
	}
	item.CalcRecipients()

	require.True(t, item.IsVisibleTo(NewAnonymousPermissions(), "https://remote.example/@recipient"))
	require.False(t, item.IsVisibleTo(NewAnonymousPermissions(), ""))
}

// FuzzOutboxItem_CalcRecipients verifies that any JSON activity calculates recipients without
// panicking, stores no empty recipient and no repeat, and answers visibility without panicking
func FuzzOutboxItem_CalcRecipients(f *testing.F) {

	f.Add(`{"actor":"https://example.com/@a","to":["https://b.example/1"],"cc":"https://b.example/2"}`, "https://b.example/1")
	f.Add(`{"actor":{"id":"https://example.com/@a"},"to":[{"id":"https://b.example/1"},{"type":"Person"}]}`, "https://example.com/@a")
	f.Add(`{"to":[[["nested"]]],"bcc":{"id":{"id":"x"}},"bto":null}`, "x")
	f.Add(`{"to":[1,true,null,{}],"cc":[""]}`, "")
	f.Add(`null`, "")

	f.Fuzz(func(t *testing.T, document string, viewer string) {

		activity := mapof.NewAny()

		if err := json.Unmarshal([]byte(document), &activity); err != nil {
			return
		}

		item := NewOutboxItem()
		item.Activity = activity
		item.CalcRecipients()

		seen := map[string]bool{}

		for _, recipient := range item.Recipients {
			require.NotEmpty(t, recipient)
			require.False(t, seen[recipient], "repeated recipient %q", recipient)
			seen[recipient] = true
		}

		// Any viewer is answered without panicking
		_ = item.IsVisibleTo(NewPermissions(), viewer)

		// RULE: An unsigned viewer never matches anything but permissions
		if viewer == "" {
			require.False(t, item.IsVisibleTo(NewPermissions(), viewer))
		}
	})
}
