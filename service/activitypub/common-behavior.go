package activitypub

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
	"github.com/go-fed/httpsig"
)

// https://go-fed.org/ref/activity/pub#The-CommonBehavior-Interface

type CommonBehavior struct {
	db *Database
}

func NewCommonBehavior(db *Database) CommonBehavior {
	return CommonBehavior{
		db: db,
	}
}

func (service CommonBehavior) AuthenticateGetInbox(ctx context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	return ctx, true, nil
}

func (service CommonBehavior) AuthenticateGetOutbox(ctx context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	return ctx, true, nil
}

func (service CommonBehavior) GetOutbox(ctx context.Context, r *http.Request) (vocab.ActivityStreamsOrderedCollectionPage, error) {
	return service.db.GetOutbox(ctx, r.URL)
}

func (service CommonBehavior) NewTransport(ctx context.Context, actorBoxIRI *url.URL, gofedAgent string) (t pub.Transport, err error) {

	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestPref := httpsig.DigestSha256
	getHeadersToSign := []string{httpsig.RequestTarget, "Date"}
	postHeadersToSign := []string{httpsig.RequestTarget, "Date", "Digest"}

	// Using github.com/go-fed/httpsig for HTTP Signatures:
	getSigner, _, err := httpsig.NewSigner(prefs, digestPref, getHeadersToSign, httpsig.Signature)
	postSigner, _, err := httpsig.NewSigner(prefs, digestPref, postHeadersToSign, httpsig.Signature)
	pubKeyId, privKey, err := s.getKeysForActorBoxIRI(actorBoxIRI)

	client := &http.Client{
		Timeout: time.Second * 30,
	}
	t = pub.NewHttpSigTransport(
		client,
		"emissary.social",
		NewClock(),
		getSigner,
		postSigner,
		pubKeyId,
		privKey)

	return
}
