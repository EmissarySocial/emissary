package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/oembed/
func GetOEmbed(serverFactory *server.Factory) func(model.Authorization, txn.GetOEmbed) (map[string]any, error) {

	return func(model.Authorization, txn.GetOEmbed) (map[string]any, error) {
		// TODO: (oembed/TODO.md Phase 9.6) Re-evaluate when the oEmbed rework lands.
		// This endpoint SERVES oEmbed for local records, so it should be built on the
		// benpate/oembed Phase 9 server primitives plus handler.GetOEmbed's record
		// resolution — not on sherlock/metadata, which is the CONSUMER-side engine.
		return map[string]any{}, derp.NotImplemented("handler.mastodon.GetOEmbed")
	}
}
