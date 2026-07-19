package activitypub_user

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// inboxActivityDocument builds a streams.Document for the suppress-storage table below.
func inboxActivityDocument(activityType string, addressing []any, object any) streams.Document {

	value := mapof.Any{
		vocab.PropertyType:  activityType,
		vocab.PropertyActor: "https://example.com/@sender",
	}

	if addressing != nil {
		value[vocab.PropertyTo] = addressing
	}

	if object != nil {
		value[vocab.PropertyObject] = object
	}

	return streams.NewDocument(value)
}

// TestInbox_SuppressStorage pins the 4B storage gate: only a MUTED actor's plain, non-public,
// non-MLS Create is suppressed. Everything else stores.
func TestInbox_SuppressStorage(t *testing.T) {

	muted := model.RuleDisposition{Action: model.RuleActionMute}
	blocked := model.RuleDisposition{Action: model.RuleActionBlock}
	clean := model.RuleDisposition{}

	privateTo := []any{"https://this.server/@recipient"}
	publicTo := []any{vocab.NamespaceActivityStreamsPublic}

	plainNote := map[string]any{vocab.PropertyType: "Note", vocab.PropertyContent: "Hello"}
	mlsNote := map[string]any{vocab.PropertyType: "Note", vocab.PropertyMediaType: vocab.MediaTypeMLS}

	// The one suppressed shape: muted + Create + non-public + non-MLS
	directMessage := inboxActivityDocument(vocab.ActivityTypeCreate, privateTo, plainNote)
	require.True(t, inbox_SuppressStorage(muted, directMessage))

	// A clean or blocked disposition never suppresses (blocked non-MLS never reaches this gate anyway)
	require.False(t, inbox_SuppressStorage(clean, directMessage))
	require.False(t, inbox_SuppressStorage(blocked, directMessage))

	// MLS is never dropped, even from a muted sender (4B)
	mlsMessage := inboxActivityDocument(vocab.ActivityTypeCreate, privateTo, mlsNote)
	require.False(t, inbox_SuppressStorage(muted, mlsMessage))

	// A muted actor's public post stores as today; the newsfeed walk hides it instead
	publicPost := inboxActivityDocument(vocab.ActivityTypeCreate, publicTo, plainNote)
	require.False(t, inbox_SuppressStorage(muted, publicPost))

	// RULE: only Create is suppressed. Likes and Undos carry no addressing (IsPublic is FALSE for
	// them), so an addressing-only test would swallow muted aggregates (R9) and subtractive
	// actions (D6).
	mutedLike := inboxActivityDocument(vocab.ActivityTypeLike, nil, "https://this.server/post/1")
	require.False(t, inbox_SuppressStorage(muted, mutedLike))

	mutedUndo := inboxActivityDocument(vocab.ActivityTypeUndo, nil, "https://example.com/likes/1")
	require.False(t, inbox_SuppressStorage(muted, mutedUndo))

	mutedDelete := inboxActivityDocument(vocab.ActivityTypeDelete, nil, "https://example.com/notes/1")
	require.False(t, inbox_SuppressStorage(muted, mutedDelete))
}
