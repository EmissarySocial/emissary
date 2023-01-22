package tasks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// SendWebSubMessage sends a WebSub notification to a single WebSub follower.
type SendWebSubMessage struct {
	stream   model.Stream
	follower model.Follower
}

func NewSendWebSubMessage(stream model.Stream, follower model.Follower) SendWebSubMessage {
	return SendWebSubMessage{
		stream:   stream,
		follower: follower,
	}
}

func (task SendWebSubMessage) Run() error {

	var body []byte

	switch task.follower.Format {

	case model.MimeTypeJSONFeed:
		// Convert & Marshall

	case model.MimeTypeAtom:
		// Convert & Marshall

	case model.MimeTypeRSS:
		// Convert & Marshall
	}

	transaction := remote.Post(task.follower.Actor.InboxURL).
		Header("Content-Type", task.follower.Format).
		Body(string(body))

	// Add HMAC signature, if necessary
	if secret, ok := task.follower.Data.GetString("secret"); ok && (secret != "") {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		transaction.Header("X-Hub-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	// Try to send the transaction to the remote WebSub client
	if err := transaction.Send(); err != nil {
		return derp.Report(derp.Wrap(err, "tasks.SendWebSubMessage", "Error sending WebSub message", task.follower))
	}

	// Woot woot!
	return nil
}
