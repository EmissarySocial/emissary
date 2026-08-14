package service

import (
	"net/http"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/streams"
)

// publicKeyLoader loads the document that carries the public key named by a keyID.
type publicKeyLoader func(keyID string, options ...any) (streams.Document, error)

// PublicKeyFinder returns the PEM-encoded public key named by a keyID, loading it through the app
// client so that the cache, the blocking rules, and the private-IP policy all apply.
func (service *ActivityStream) PublicKeyFinder(keyID string) (string, error) {
	return findPublicKey(service.loadPublicKey, keyID)
}

// RefreshPublicKey re-fetches an Actor's public key after a signature has already failed against the
// copy that PublicKeyFinder returned.
func (service *ActivityStream) RefreshPublicKey(keyID string) (string, error) {
	return refreshPublicKey(service.loadPublicKey, keyID)
}

// VerifySignature verifies the HTTP Signature attached to an inbound request, and returns the
// Signature that identifies the Actor who signed it.
func (service *ActivityStream) VerifySignature(request *http.Request) (sigs.Signature, error) {

	const location = "service.ActivityStream.VerifySignature"

	signature, err := sigs.Verify(request, service.PublicKeyFinder, sigs.WithRefreshKey(service.RefreshPublicKey))

	if err != nil {
		return signature, derp.Wrap(err, location, "Verifying HTTP Signature")
	}

	return signature, nil
}

// loadPublicKey returns the Actor document that carries the public key named by a keyID.
func (service *ActivityStream) loadPublicKey(keyID string, options ...any) (streams.Document, error) {

	const location = "service.ActivityStream.loadPublicKey"

	// This works because the ashash client resolves the keyID's "#fragment" against the owning
	// Actor's JSON-LD document.
	publicKey, err := service.AppClient().Load(keyID, options...)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Loading public key", keyID)
	}

	return publicKey, nil
}

//////////////////////////////////////////
// Helper functions
//////////////////////////////////////////

// findPublicKey returns the PEM certificate carried by the document a loader produces.
// separated for testability.
func findPublicKey(load publicKeyLoader, keyID string) (string, error) {

	const location = "service.findPublicKey"

	publicKey, err := load(keyID)

	if err != nil {
		return "", derp.Wrap(err, location, "Loading public key", keyID)
	}

	return publicKey.PublicKeyPEM(), nil
}

// refreshPublicKey reloads a public key, bypassing the cached copy that a
// signature has already failed against. separated for testability.
func refreshPublicKey(load publicKeyLoader, keyID string) (string, error) {

	const location = "service.refreshPublicKey"

	// A failure against a cached key is the signal that the remote may have rotated it. Refreshing on
	// that signal -- rather than on a timer -- repairs rotation on the first rejected delivery, with no
	// window in which an honest peer's traffic bounces. ascache's default cooldown is what keeps the
	// same signal from becoming an amplifier: a forged signature naming any keyId host would otherwise
	// buy the sender one outbound fetch per request, aimed wherever they chose. (BUG-22)
	publicKey, err := load(keyID, ascache.WithWriteOnly())

	if err != nil {
		return "", derp.Wrap(err, location, "Reloading public key", keyID)
	}

	return publicKey.PublicKeyPEM(), nil
}
