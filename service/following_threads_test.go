package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

func TestGetPrimaryDocument_1(t *testing.T) {

	original := streams.NewDocument(map[string]any{
		vocab.PropertyID: "https://document-1.com/",
	})

	primary, originType := getPrimaryPost(original, model.OriginTypePrimary)

	require.Equal(t, model.OriginTypePrimary, originType)
	require.Equal(t, "https://document-1.com/", primary.ID())
}

func TestGetPrimaryDocument_2(t *testing.T) {

	original := streams.NewDocument(map[string]any{
		vocab.PropertyID: "https://document-1.com/",
		vocab.PropertyInReplyTo: map[string]any{
			vocab.PropertyID: "https://document-2.com/",
		},
	})

	primary, originType := getPrimaryPost(original, model.OriginTypePrimary)

	require.Equal(t, model.OriginTypeReply, originType)
	require.Equal(t, "https://document-2.com/", primary.ID())
}
