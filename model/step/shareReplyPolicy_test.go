package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestNewShareReplyPolicy pins that the step reads its Template, and refuses a definition without one
func TestNewShareReplyPolicy(t *testing.T) {

	result, err := NewShareReplyPolicy(mapof.Any{"template": "outbox-message"})
	require.Nil(t, err)
	require.Equal(t, "outbox-message", result.TemplateID)
	require.Equal(t, "share-reply-policy", result.Name())
	require.Equal(t, "Stream", result.RequiredModel())

	_, err = NewShareReplyPolicy(mapof.Any{})
	require.NotNil(t, err)
}

// TestNew_ShareReplyPolicy pins that the step is registered under its name
func TestNew_ShareReplyPolicy(t *testing.T) {

	result, err := New(mapof.Any{"do": "share-reply-policy", "template": "outbox-message"}, nil)
	require.Nil(t, err)
	require.IsType(t, ShareReplyPolicy{}, result)
}
