package providers

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestNull_LifecycleIsAlwaysSuccessful confirms all four lifecycle methods return nil
func TestNull_LifecycleIsAlwaysSuccessful(t *testing.T) {

	adapter := Null{}
	connection := model.NewConnection()
	vault := mapof.NewString()

	require.NoError(t, adapter.BeforeSave(&connection, vault))
	require.NoError(t, adapter.Connect(&connection, vault, "example.com"))
	require.NoError(t, adapter.Refresh(&connection, vault))
	require.NoError(t, adapter.Disconnect(&connection, vault))
}

// TestNull_LifecycleLeavesTheConnectionAlone confirms Null writes nothing to the Connection
func TestNull_LifecycleLeavesTheConnectionAlone(t *testing.T) {

	adapter := Null{}
	vault := mapof.NewString()

	connection := model.NewConnection()
	connection.Active = true
	connection.ProviderID = "TEST-PROVIDER"
	connection.Data.SetString("sentinel", "untouched")

	before := connection

	require.NoError(t, adapter.BeforeSave(&connection, vault))
	require.NoError(t, adapter.Connect(&connection, vault, "example.com"))
	require.NoError(t, adapter.Refresh(&connection, vault))
	require.NoError(t, adapter.Disconnect(&connection, vault))

	require.Equal(t, before, connection)
	require.True(t, connection.Active)
	require.Equal(t, "untouched", connection.Data.GetString("sentinel"))
}

// TestNull_LifecycleAcceptsNilConnection confirms Null never dereferences its arguments,
// which is what makes it safe as service.Provider's fallback for an unknown providerID.
func TestNull_LifecycleAcceptsNilConnection(t *testing.T) {

	adapter := Null{}

	require.NotPanics(t, func() {
		require.NoError(t, adapter.BeforeSave(nil, nil))
		require.NoError(t, adapter.Connect(nil, nil, ""))
		require.NoError(t, adapter.Refresh(nil, nil))
		require.NoError(t, adapter.Disconnect(nil, nil))
	})
}

// TestNull_PollStreamsReturnsANilChannel pins the fact that PollStreams returns nil rather
// than a closed channel, so a caller must test for nil before ranging over it.
func TestNull_PollStreamsReturnsANilChannel(t *testing.T) {

	connection := model.NewConnection()
	channel := Null{}.PollStreams(&connection)

	require.Nil(t, channel)

	// RULE: A nil channel is never ready, so a select with a default falls through.
	// Ranging over one would block this goroutine forever.
	select {
	case <-channel:
		t.Fatal("a nil channel must never be ready")
	default:
	}
}

// TestNull_PollStreamsAcceptsNilConnection confirms PollStreams ignores its argument
func TestNull_PollStreamsAcceptsNilConnection(t *testing.T) {

	require.NotPanics(t, func() {
		require.Nil(t, Null{}.PollStreams(nil))
	})
}
