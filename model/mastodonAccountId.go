package model

import (
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/benpate/toot/object"
)

// remoteAccountIDPrefix marks a Mastodon account ID that encodes a remote actor's
// URL rather than a local hex ObjectID. "u_" can never begin a 24-char hex
// ObjectID, so a prefixed value is unambiguously a remote ID.
const remoteAccountIDPrefix = "u_"

// EncodeRemoteAccountID turns a remote actor's URL into an opaque, URL-path-safe
// Mastodon account ID. The ID is a pure function of the URL, so it stays stable
// even though the ascache row backing the actor is reminted on every refetch.
func EncodeRemoteAccountID(actorURL string) string {
	return remoteAccountIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(actorURL))
}

// DecodeRemoteAccountID reverses EncodeRemoteAccountID. ok is false when s is not
// one of our encoded remote IDs (no prefix, not valid base64url, or the decoded
// value is not an absolute URL), so callers fall through to local-hex / bare-URL
// handling.
func DecodeRemoteAccountID(s string) (actorURL string, ok bool) {

	if !strings.HasPrefix(s, remoteAccountIDPrefix) {
		return "", false
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, remoteAccountIDPrefix))

	if err != nil {
		return "", false
	}

	if parsed, err := url.Parse(string(raw)); err != nil || !parsed.IsAbs() {
		return "", false
	}

	return string(raw), true
}

// RemoteActorAccount builds a Mastodon Account for a remote actor from the
// minimal data Emissary keeps about one it has not fully dereferenced -- a
// Following row, a news-feed origin. profileURL is required; label and iconURL
// are best-effort. The "acct" (user@domain) is derived from the URL, and the
// account ID is the encoded URL so it matches every other surface.
func RemoteActorAccount(profileURL string, label string, iconURL string, createdAt time.Time) object.Account {

	username := label
	acct := label

	if parsed, err := url.Parse(profileURL); err == nil && parsed.Host != "" {

		name := strings.Trim(parsed.Path, "/")

		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}

		name = strings.TrimPrefix(name, "@")

		if name != "" {
			username = name
			acct = name + "@" + parsed.Host
		}
	}

	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return object.Account{
		ID:          EncodeRemoteAccountID(profileURL),
		Username:    username,
		Acct:        acct,
		URL:         profileURL,
		DisplayName: label,
		Avatar:      iconURL,
		CreatedAt:   MastodonDate(createdAt),
	}
}
