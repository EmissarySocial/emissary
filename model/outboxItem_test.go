package model

import (
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
