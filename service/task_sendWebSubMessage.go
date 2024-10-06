package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
)

// TaskSendWebSubMessage sends a WebSub notification to a single WebSub follower.
type TaskSendWebSubMessage struct {
	Follower model.Follower
}

func NewTaskSendWebSubMessage(follower model.Follower) TaskSendWebSubMessage {
	return TaskSendWebSubMessage{
		Follower: follower,
	}
}

func (task TaskSendWebSubMessage) Priority() int {
	return 20
}

func (task TaskSendWebSubMessage) RetryMax() int {
	return 12 // 4096 minutes = 68 hours ~= 3 days
}

func (task TaskSendWebSubMessage) MarshalMap() map[string]any {
	return mapof.Any{
		"follower": task.Follower,
	}
}

func (task TaskSendWebSubMessage) Hostname() string {
	return domain.NameOnly(task.Follower.Actor.InboxURL)
}

func (task TaskSendWebSubMessage) Run() error {

	var body []byte

	// TODO: LOW: SendWebSubMessage will require a refactor if we want to send "fat pings":
	// https://indieweb.org/How_to_publish_and_consume_WebSub
	switch task.Follower.Format {

	case model.MimeTypeJSONFeed:
		// Convert & Marshall

	case model.MimeTypeAtom:
		// Convert & Marshall

	case model.MimeTypeRSS:
		// Convert & Marshall
	}

	transaction := remote.Post(task.Follower.Actor.InboxURL).
		Header("Content-Type", task.Follower.Format).
		Body(string(body))

	// Add HMAC signature, if necessary
	if secret := task.Follower.Data.GetString("secret"); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		transaction.Header("X-Hub-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	// Try to send the transaction to the remote WebSub client
	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, "service.TaskSendWebSubMessage", "Error sending WebSub message", task.Follower)
	}

	// Woot woot!
	return nil
}
