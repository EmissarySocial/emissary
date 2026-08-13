package service

import (
	"net/http"
	"time"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/streams"
)

// publicKeyCooldown is the minimum interval between forced reloads of a single Actor's public key.
const publicKeyCooldown = 60 * time.Second

// publicKeyLoader loads the document that carries the public key named by a keyID.
type publicKeyLoader func(keyID string, options ...any) (streams.Document, error)

// VerifySignature verifies the HTTP Signature attached to an inbound request, and returns the
// Signature that identifies the Actor who signed it.
func (service *ActivityStream) VerifySignature(request *http.Request) (sigs.Signature, error) {

	// This is the ONE funnel for inbound signature verification -- inboxes, authorized fetch, and
	// permissions all arrive here. There is deliberately no exported single-shot key finder to reach
	// around it, because a caller that used one would silently opt out of the refresh below.
	return verifySignature(request, service.loadPublicKey)
}

// loadPublicKey returns the Actor's public key named by a keyID, loading it through the app client.
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

// verifySignature verifies an inbound request, refreshing the signer's key when a cached key is the
// plausible cause of a failure.
func verifySignature(request *http.Request, loadPublicKey publicKeyLoader) (sigs.Signature, error) {

	const location = "service.verifySignature"

	// Verify against the key we already hold.  For the overwhelming majority of requests this is the
	// only pass, and a cached Actor makes it cost no network at all.
	finder := newPublicKeyFinder(loadPublicKey)
	signature, err := sigs.Verify(request, finder.find)

	if err == nil {
		return signature, nil
	}

	// RULE: Only a key that CAME FROM the cache can be out of date.  If the finder was never reached
	// (a bad digest, an expired date, an unparseable signature) or it already spoke to the origin,
	// then asking again would put the same question to the same server.
	if finder.notRefreshable() {
		return signature, derp.Wrap(err, location, "Verifying HTTP Signature (fresh actor JSON)")
	}

	// A failure against a CACHED key is the signal that the remote may have rotated it. Refreshing on
	// that signal -- rather than on a timer -- repairs rotation on the first rejected delivery, with
	// no window in which an honest peer's traffic bounces. WithMinAge is what keeps that same signal
	// from becoming an amplifier: a forged signature naming any keyId host would otherwise buy the
	// sender one outbound fetch per request, aimed wherever they chose. (BUG-22 D1, D3)
	refreshed, refreshErr := loadPublicKey(finder.keyID, ascache.WithWriteOnly(), ascache.WithMinAge(publicKeyCooldown))

	// A reload that fails tells us nothing the first pass did not, so the original verification error
	// stands.  It is dropped on purpose: for a forged keyId this path fails as a matter of course, and
	// reporting it would let a sender fill the log.
	if refreshErr != nil {
		return signature, derp.Wrap(err, location, "Verifying HTTP Signature (could not reload actor)")
	}

	// RULE: An unchanged key means the signature is simply bad.  Re-running the same verification
	// against the same key would only produce the same error, one CPU-bound crypto check later.
	rotatedPEM := refreshed.PublicKeyPEM()

	if rotatedPEM == finder.publicKeyPEM {
		return signature, derp.Wrap(err, location, "Verifying HTTP Signature (key is unchanged)")
	}

	// The remote HAS rotated.  Verify once more, against the key they are using now.
	signature, err = sigs.Verify(request, rotatedKeyFinder(finder.keyID, rotatedPEM))

	if err != nil {
		return signature, derp.Wrap(err, location, "Verifying HTTP Signature against rotated key")
	}

	// The bearer of this signature may pass.
	return signature, nil
}

// rotatedKeyFinder returns a sigs.PublicKeyFinder that serves one already-loaded key, and serves it
// only to the keyID it was loaded for.
func rotatedKeyFinder(keyID string, publicKeyPEM string) sigs.PublicKeyFinder {

	const location = "service.rotatedKeyFinder"

	// The keyID check is belt and suspenders: the second pass re-parses the same request, so it can
	// only ask for the same keyID. Were that ever to change, serving this key regardless would
	// authenticate a signature against a key its own keyId never named.
	return func(requestedKeyID string) (string, error) {

		if requestedKeyID != keyID {
			return "", derp.Internal(location, "Signature requested an unexpected keyID", requestedKeyID, keyID)
		}

		return publicKeyPEM, nil
	}
}
