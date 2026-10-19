package asnormalizer

import (
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// propertyVisibility is the key a post's audience is stored under, as one of the Mastodon words
// "public", "unlisted", "private" or "direct". ActivityPub has no property of its own for this.
const propertyVisibility = "visibility"

// Visibility works out who can see a post from its addressing, as the Mastodon word public, unlisted,
// private or direct, and returns "" when the post names no audience at all.
func Visibility(document streams.Document, followersURL string) string {

	to := addressees(document.To())
	cc := addressees(document.CC())

	if len(to) == 0 && len(cc) == 0 {
		return ""
	}

	// Public in "to" is public, in "cc" is unlisted, the author's followers is followers-only, anyone else is direct
	if containsPublic(to) {
		return "public"
	}

	if containsPublic(cc) {
		return "unlisted"
	}

	for _, id := range append(to, cc...) {
		if isFollowersCollection(id, followersURL) {
			return "private"
		}
	}

	return "direct"
}

// addressees lists the IDs in one addressing property ("to" or "cc").
func addressees(document streams.Document) []string {

	result := []string{}

	for id := range document.RangeIDs() {
		if id != "" {
			result = append(result, id)
		}
	}

	return result
}

// containsPublic returns TRUE if the list names the Public collection, in any spelling in use.
func containsPublic(ids []string) bool {

	for _, id := range ids {
		switch id {
		case vocab.NamespaceActivityStreamsPublic, vocab.NamespaceASPublic, "Public":
			return true
		}
	}

	return false
}

// isFollowersCollection returns TRUE if the ID is the author's followers collection: the known address
// when we have one, otherwise any address that ends in "/followers", as Mastodon and Pleroma use.
func isFollowersCollection(id string, followersURL string) bool {

	if followersURL != "" && id == followersURL {
		return true
	}

	return strings.HasSuffix(id, "/followers")
}
