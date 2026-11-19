package mastodon

import (
	"encoding/json"
	"testing"

	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// TestStatusEdit_DatesAndContent confirms a revision copies the status and is dated by its last edit when there is one.
func TestStatusEdit_DatesAndContent(t *testing.T) {

	status := object.Status{
		Content:     "<p>hello</p>",
		SpoilerText: "cw",
		Sensitive:   true,
		CreatedAt:   "2026-10-01T10:00:00.000Z",
		Account:     object.Account{ID: "author"},
	}

	edit := statusEdit(status)
	require.Equal(t, "<p>hello</p>", edit.Content)
	require.Equal(t, "cw", edit.SpoilerText)
	require.True(t, edit.Sensitive)
	require.Equal(t, "author", edit.Account.ID)
	require.Equal(t, "2026-10-01T10:00:00.000Z", edit.CreatedAt, "an unedited status is dated by its creation")

	status.EditedAt = "2026-10-02T11:00:00.000Z"
	require.Equal(t, "2026-10-02T11:00:00.000Z", statusEdit(status).CreatedAt, "an edited status is dated by its last edit")
}

// TestStatusEdit_EmptyListsAreArrays confirms a status with no media or emoji still sends JSON arrays, never null.
func TestStatusEdit_EmptyListsAreArrays(t *testing.T) {

	data, err := json.Marshal(statusEdit(object.Status{}))
	require.NoError(t, err)
	require.Contains(t, string(data), `"media_attachments":[]`)
	require.Contains(t, string(data), `"emojis":[]`)
}
