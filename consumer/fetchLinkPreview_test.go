package consumer

import (
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// TestFetchLinkPreview_Priority confirms a page read waits in storage, behind work a person is watching
func TestFetchLinkPreview_Priority(t *testing.T) {

	task := queue.NewTask(service.TaskFetchLinkPreview, mapof.Any{"hostname": "example.com", "url": "https://news.example/"})
	task.Priority = -1

	require.NoError(t, PreProcessor(&task))

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1 sentinel.
	require.Equal(t, 64, task.Priority)
}

// TestFetchLinkPreview_RefusesBadLinks confirms a task whose link can never be read fails at once
func TestFetchLinkPreview_RefusesBadLinks(t *testing.T) {

	for _, link := range []string{"", "/relative", "javascript:alert(1)"} {
		result := FetchLinkPreview(nil, mapof.Any{"url": link})
		require.Equal(t, queue.ResultStatusFailure, result.Status, link)
	}
}
