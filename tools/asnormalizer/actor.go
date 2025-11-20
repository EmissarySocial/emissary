package asnormalizer

import (
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// Actor normalizes an Actor document
func Actor(document streams.Document) map[string]any {

	result := map[string]any{

		// Profile
		vocab.PropertyType:              document.Type(),
		vocab.PropertyID:                document.ID(),
		vocab.PropertyName:              document.Name(),
		vocab.PropertyPreferredUsername: document.PreferredUsername(),
		vocab.PropertySummary:           document.Summary(),
		vocab.PropertyImage:             Image(document.Image()),
		vocab.PropertyIcon:              Image(document.Icon()),
		vocab.PropertyTag:               Tags(document.Tag()),
		vocab.PropertyURL:               document.URL(),
		vocab.PropertyEndpoints:         document.Endpoints().Value(),

		// Collections
		vocab.PropertyAttachment: document.Attachment().Value(),
		vocab.PropertyInbox:      document.Inbox().String(),
		vocab.PropertyOutbox:     document.Outbox().Value(), // using raw outbox value because RSS feeds stuff data in here.
		vocab.PropertyLiked:      document.Liked().String(),
		vocab.PropertyFeatured:   document.Featured().String(),
		vocab.PropertyFollowers:  document.Followers().String(),
		vocab.PropertyFollowing:  document.Following().String(),
		vocab.PropertyPublicKey:  document.PublicKey().Value(),
	}

	// Cryptography
	if publicKey := document.PublicKey(); publicKey.NotNil() {
		result[vocab.PropertyPublicKey] = map[string]any{
			vocab.PropertyID:           publicKey.ID(),
			vocab.PropertyPublicKeyPEM: publicKey.PublicKeyPEM(),
		}
	}

	return result
}

// ActorSummary normalizes an Actor document to be embedded in other documents
func ActorSummary(document streams.Document) map[string]any {

	result := map[string]any{

		// Profile
		vocab.PropertyType:              document.Type(),
		vocab.PropertyID:                document.ID(),
		vocab.PropertyName:              document.Name(),
		vocab.PropertyPreferredUsername: document.PreferredUsername(),
		vocab.PropertySummary:           document.Summary(),
		vocab.PropertyImage:             Image(document.Image()),
		vocab.PropertyIcon:              Image(document.Icon()),
		vocab.PropertyTag:               Tags(document.Tag()),
		vocab.PropertyURL:               document.URL(),
	}

	return result
}
