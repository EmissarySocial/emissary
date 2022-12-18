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

	// TODO: MEDIUM: Add encoding for the message body

	transaction := remote.Post(task.follower.Actor.InboxURL).
		Header("Content-Type", "application/json").
		// Header("Link", `<`+task.stream.Document.URL+`/websub; rel="hub", <`+task.stream.Document.URL+`>; rel="self"`).
		Body(string(body))

	// Add HMAC signature, if necessary
	if secret := task.follower.Data.GetString("secret"); secret != "" {
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
