package consumer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func SendWebSubMessage(args mapof.Any) queue.Result {

	const location = "consumer.SendWebSubMessage"

	// Collect task parameters
	inboxURL := args.GetString("inboxUrl")
	format := args.GetString("format")
	secret := args.GetString("secret") // nolint:scopeguard

	var body []byte

	// TODO: LOW: SendWebSubMessage will require a refactor if we want to send "fat pings":
	// https://indieweb.org/How_to_publish_and_consume_WebSub
	switch format {

	case model.MimeTypeJSONFeed:
		// Convert & Marshall

	case model.MimeTypeAtom:
		// Convert & Marshall

	case model.MimeTypeRSS:
		// Convert & Marshall
	}

	transaction := remote.Post(inboxURL).
		Header("Content-Type", format).
		Body(string(body))

	// Add HMAC signature, if necessary
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		transaction.Header("X-Hub-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	// Try to send the transaction to the remote WebSub client
	if err := transaction.Send(); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to send WebSub message"))
	}

	// Woot woot!
	return queue.Success()
}
