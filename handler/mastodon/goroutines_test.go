package mastodon

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

// panickingClient serves documents like fakeClient, but panics when asked for one address, the way
// a hostile document can panic the code that reads it.
type panickingClient struct {
	*fakeClient
	panicOn string
}

// Load panics for the hostile address, and serves every other one from the fake client.
func (client *panickingClient) Load(uri string, options ...any) (streams.Document, error) {

	if uri == client.panicOn {
		panic("hostile document")
	}

	return client.fakeClient.Load(uri, options...)
}

// TestRecoverGoroutine_StopsAPanic pins that a panic in a goroutine is reported rather than fatal.
// Without the recover, this test would end the whole test binary.
func TestRecoverGoroutine_StopsAPanic(t *testing.T) {

	done := make(chan struct{})

	go func() {
		defer close(done)
		defer recoverGoroutine("handler.mastodon.TestRecoverGoroutine_StopsAPanic")
		panic("boom")
	}()

	<-done
}

// TestRecoverGoroutine_NoPanic pins that the deferred call is harmless when nothing panics.
func TestRecoverGoroutine_NoPanic(t *testing.T) {

	finished := false

	func() {
		defer recoverGoroutine("handler.mastodon.TestRecoverGoroutine_NoPanic")
		finished = true
	}()

	require.True(t, finished)
}

// TestLoadThreadDocuments_SurvivesAPanickingLoad pins that one document that panics while loading
// is left out of a thread, while the rest of the thread still loads.
func TestLoadThreadDocuments_SurvivesAPanickingLoad(t *testing.T) {

	client := &panickingClient{
		fakeClient: &fakeClient{documents: map[string]map[string]any{
			"https://a.example/notes/1": {"id": "https://a.example/notes/1", "type": "Note"},
		}},
		panicOn: "https://hostile.example/notes/2",
	}

	result := loadThreadDocuments(client, []string{"https://a.example/notes/1", "https://hostile.example/notes/2"})

	require.Len(t, result, 1)
	require.Equal(t, "https://a.example/notes/1", result["https://a.example/notes/1"].ID())
}
