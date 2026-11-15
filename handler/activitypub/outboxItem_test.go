package activitypub

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// fixedSignature is a SignatureParser that answers every request with the same viewer
type fixedSignature struct {
	permissions model.Permissions
	actorURL    string
	err         error
}

// ParseHTTPSignatureWithActor returns the viewer this parser was built with
func (parser fixedSignature) ParseHTTPSignatureWithActor(data.Session, *http.Request) (model.Permissions, string, error) {
	return parser.permissions, parser.actorURL, parser.err
}

// directMessage returns an outbox item that only its author and one recipient may read
func directMessage() model.OutboxItem {

	item := model.NewOutboxItem()
	item.Activity = mapof.Any{
		vocab.PropertyID:    "https://example.com/@author/pub/outbox/" + item.ActivityID.Hex(),
		vocab.PropertyActor: "https://example.com/@author",
		vocab.PropertyTo:    []string{"https://remote.example/@recipient"},
		vocab.PropertyBCC:   []string{"https://remote.example/@blind"},
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyContent: "secret",
			vocab.PropertyBTo:     []string{"https://remote.example/@other"},
		},
	}
	item.CalcRecipients()

	return item
}

// TestVisibleOutboxItem_HiddenIsNotFound verifies that a viewer the item does not admit sees a 404
func TestVisibleOutboxItem_HiddenIsNotFound(t *testing.T) {

	item := directMessage()

	for name, parser := range map[string]fixedSignature{
		"unsigned": {permissions: model.NewAnonymousPermissions()},
		"stranger": {permissions: model.NewAnonymousPermissions(), actorURL: "https://remote.example/@stranger"},
	} {
		result, err := VisibleOutboxItem(parser, nil, nil, &item)
		require.Error(t, err, name)
		require.True(t, derp.IsNotFound(err), name)
		require.Nil(t, result, name)
	}
}

// TestVisibleOutboxItem_RecipientReadsWithoutBlindFields verifies that a named recipient reads the
// activity, with `bto` and `bcc` removed at every level
func TestVisibleOutboxItem_RecipientReadsWithoutBlindFields(t *testing.T) {

	item := directMessage()
	parser := fixedSignature{permissions: model.NewAnonymousPermissions(), actorURL: "https://remote.example/@recipient"}

	result, err := VisibleOutboxItem(parser, nil, nil, &item)
	require.NoError(t, err)
	require.Equal(t, item.Activity[vocab.PropertyID], result[vocab.PropertyID])
	require.NotContains(t, result, vocab.PropertyBCC)
	require.NotContains(t, result[vocab.PropertyObject], vocab.PropertyBTo)

	// The stored activity keeps its blind recipients, because delivery reads them
	require.Contains(t, item.Activity, vocab.PropertyBCC)
}

// TestVisibleOutboxItem_PublicIsOpen verifies that an unsigned viewer reads an item with anonymous permissions
func TestVisibleOutboxItem_PublicIsOpen(t *testing.T) {

	item := directMessage()
	item.Permissions = model.NewAnonymousPermissions()

	result, err := VisibleOutboxItem(fixedSignature{permissions: model.NewAnonymousPermissions()}, nil, nil, &item)
	require.NoError(t, err)
	require.NotEmpty(t, result)
}

// TestVisibleOutboxItem_InvalidSignatureIsRefused verifies that a signature error is returned as it is
func TestVisibleOutboxItem_InvalidSignatureIsRefused(t *testing.T) {

	item := directMessage()
	item.Permissions = model.NewAnonymousPermissions()
	refusal := derp.Unauthorized("test", "Invalid HTTP Signature")

	result, err := VisibleOutboxItem(fixedSignature{err: refusal}, nil, nil, &item)
	require.Equal(t, refusal, err)
	require.Nil(t, result)
}
