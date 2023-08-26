package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// TaskSendWebSubMessage sends a WebSub notification to a single WebSub follower.
type TaskSendWebSubMessage struct {
	follower model.Follower
}

func NewTaskSendWebSubMessage(follower model.Follower) TaskSendWebSubMessage {
	return TaskSendWebSubMessage{
		follower: follower,
	}
}

func (task TaskSendWebSubMessage) Run() error {

	var body []byte

	// TODO: LOW: SendWebSubMessage will require a refactor if we want to send "fat pings":
	// https://indieweb.org/How_to_publish_and_consume_WebSub
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
	if secret := task.follower.Data.GetString("secret"); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		transaction.Header("X-Hub-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	// Try to send the transaction to the remote WebSub client
	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, "service.TaskSendWebSubMessage", "Error sending WebSub message", task.follower)
	}

	// Woot woot!
	return nil
}
