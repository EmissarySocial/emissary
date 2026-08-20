package activitypub

import (
	"crypto/sha256"
	"encoding/base64"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/uri"
)

// IsSameOrigin returns TRUE if two URLs share the same scheme+host authority.
// An unparseable or hostless value on either side is never same-origin -- two
// empty hosts must not compare equal, or garbage-in would silently authorize.
func IsSameOrigin(first string, second string) bool {

	firstHost := uri.Host(first)

	// RULE: A value with no recognizable host binds to nothing
	if firstHost == "" {
		return false
	}

	return firstHost == uri.Host(second)
}

// FakeActivityID returns a stable hash that identifies an activity, even when it carries no ID of its own
func FakeActivityID(activity streams.Document) string {

	// Dig past create/update/delete Activities to find the real object
	switch activity.Type() {
	case vocab.ActivityTypeCreate, vocab.ActivityTypeUpdate, vocab.ActivityTypeDelete:
		activity = activity.Object()
	}

	// Use ID if it exists, otherwise hash the type, actor, and object
	if id := activity.ID(); id != "" {
		return sha256base64(id)
	}

	// Otherwise, hash the type, actor, and object
	return sha256base64(activity.ActorID() + ":" + activity.Type() + ":" + activity.Object().ID())
}

// sha256base64 returns the SHA256 hash of the input string, encoded as a base64 string
func sha256base64(value string) string {

	hash := sha256.New()
	hash.Write([]byte(value))

	return "sha-" + base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
