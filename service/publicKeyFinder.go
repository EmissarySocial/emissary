package service

import (
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
)

// publicKeyFinder is a single-use sigs.PublicKeyFinder that remembers what it handed to the verifier,
// so that a failed verification can tell a stale cached key apart from a freshly fetched one.
type publicKeyFinder struct {
	loadPublicKey publicKeyLoader // Resolves a keyID into the document that carries the key
	keyID         string          // keyID the Signature asked for, or empty if this finder was never called
	publicKeyPEM  string          // PEM certificate that was returned to the verifier
	fromCache     bool            // TRUE if the owning Actor document came from the cache rather than its origin
}

// newPublicKeyFinder returns a publicKeyFinder bound to a key loader.
func newPublicKeyFinder(loadPublicKey publicKeyLoader) *publicKeyFinder {
	return &publicKeyFinder{
		loadPublicKey: loadPublicKey,
	}
}

// find implements sigs.PublicKeyFinder, returning the PEM-encoded public key for a keyID.
func (finder *publicKeyFinder) find(keyID string) (string, error) {

	const location = "service.publicKeyFinder.find"

	// Recorded before the load, so that a FAILED load is still distinguishable from never having
	// been called at all.
	finder.keyID = keyID

	publicKey, err := finder.loadPublicKey(keyID)

	if err != nil {
		return "", derp.Wrap(err, location, "Loading public key", keyID)
	}

	// ascache stamps its own answers, and ashash carries the Actor's headers down onto the key
	// sub-document, so this reads the provenance of the fetch that produced this key.
	finder.fromCache = ascache.FromCache(publicKey)
	finder.publicKeyPEM = publicKey.PublicKeyPEM()

	return finder.publicKeyPEM, nil
}

// isRefreshable returns TRUE if this finder returned a cached key, which a reload could still change.
func (finder *publicKeyFinder) isRefreshable() bool {

	if finder.publicKeyPEM == "" {
		return false
	}

	return finder.fromCache
}

// notRefreshable returns TRUE if reloading this finder's key could not produce a different answer.
func (finder *publicKeyFinder) notRefreshable() bool {
	return !finder.isRefreshable()
}
