package gofed

import (
	"context"
	"crypto"
	"net/http"
	"net/url"
	"time"

	"github.com/benpate/derp"
	"github.com/go-fed/activity/pub"
	"github.com/go-fed/httpsig"
)

// Returns a new pub.Transport for federating with peer software. There is a pub.HttpSigTransport implementation
// provided for using HTTP and HTTP Signatures, but providing a different transport allows federating using
// different protocols.
//
// The actorBoxIRI will be either the Inbox or Outbox of an Actor who is attempting to do the dereferencing or
// delivery. Any authentication scheme applied on the request must be based on this actor. The request must
// contain some sort of credential of the user, such as a HTTP Signature.
//
// The gofedAgent passed in should be used by the pub.Transport implementation in the User-Agent, as well as the
// application-specific user agent string. The gofedAgent will indicate this library's use as well as the
// library's version number.
//
// Any server-wide rate-limiting that needs to occur should happen in a pub.Transport implementation. This
// factory function allows this to be created, so peer servers are not DOS'd.
//
// Any retry logic should also be handled by the pub.Transport implementation.
//
// Note that the library will not maintain a long-lived pointer to the returned pub.Transport so that any private
// credentials are able to be garbage collected.
//
// For more information, see the Transports section at https://go-fed.org/ref/activity/pub#Transports
func (common Common) NewTransport(c context.Context, actorBoxIRI *url.URL, gofedAgent string) (pub.Transport, error) {

	const location = "gofed.Common.NewTransport"

	// Set up Encryption config
	prefs := []httpsig.Algorithm{httpsig.RSA_SHA512}
	digestPref := httpsig.DigestAlgorithm(httpsig.DigestSha512)
	getHeadersToSign := []string{httpsig.RequestTarget, "Date"}
	postHeadersToSign := []string{httpsig.RequestTarget, "Date", "Digest"}
	expiresIn := int64(3600) // 1 hour

	// Create HTTP Signature signers
	getSigner, _, err := httpsig.NewSigner(prefs, digestPref, getHeadersToSign, httpsig.Signature, expiresIn)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error creating 'GET' signer")
	}

	postSigner, _, err := httpsig.NewSigner(prefs, digestPref, postHeadersToSign, httpsig.Signature, expiresIn)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error creating 'POST' signer")
	}

	// Retrieve public/private keys from the database
	pubKeyID, privKey, err := common.getKeysForActorBoxIRI(actorBoxIRI)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error getting public/private keys")
	}

	// Create HTTP CLient
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	return pub.NewHttpSigTransport(
		client,
		"emissary.social",
		NewClock(),
		getSigner,
		postSigner,
		pubKeyID,
		privKey), nil
}

func (common Common) getKeysForActorBoxIRI(actorIRI *url.URL) (string, crypto.PrivateKey, error) {
	return "", nil, nil
}
