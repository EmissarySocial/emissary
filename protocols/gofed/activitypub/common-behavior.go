package activitypub

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/protocols/gofed/common"
	"github.com/EmissarySocial/emissary/protocols/gofed/db"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
	"github.com/go-fed/httpsig"
)

// https://go-fed.org/ref/activity/pub#The-CommonBehavior-Interface

type CommonBehavior struct {
	db                   *db.Database
	userService          *service.User
	encryptionKeyService *service.EncryptionKey
	host                 string
}

func NewCommonBehavior(db *db.Database, userService *service.User, encryptionKeyService *service.EncryptionKey, host string) CommonBehavior {
	return CommonBehavior{
		db:                   db,
		userService:          userService,
		encryptionKeyService: encryptionKeyService,
		host:                 host,
	}
}

func (service CommonBehavior) AuthenticateGetInbox(ctx context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	return ctx, false, nil
}

func (service CommonBehavior) AuthenticateGetOutbox(ctx context.Context, w http.ResponseWriter, r *http.Request) (out context.Context, authenticated bool, err error) {
	return ctx, false, nil
}

func (service CommonBehavior) GetOutbox(ctx context.Context, r *http.Request) (vocab.ActivityStreamsOrderedCollectionPage, error) {
	return service.db.GetOutbox(ctx, r.URL)
}

func (service CommonBehavior) NewTransport(ctx context.Context, actorBoxIRI *url.URL, gofedAgent string) (t pub.Transport, err error) {

	// Get the userID from the actorBoxIRI
	userID, _, _, err := common.ParseURL(actorBoxIRI)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.activitypub.CommonBehavior.NewTransport", "Error parsing actor URL", actorBoxIRI)
	}

	// Load the user from the database
	user := model.NewUser()
	if err := service.userService.LoadByID(userID, &user); err != nil {
		return nil, derp.Wrap(err, "gofed.activitypub.CommonBehavior.NewTransport", "Error loading user", userID)
	}

	// Build the Transport interface
	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestPref := httpsig.DigestSha256
	getHeadersToSign := []string{httpsig.RequestTarget, "Date"}
	postHeadersToSign := []string{httpsig.RequestTarget, "Date", "Digest"}

	getSigner, _, err := httpsig.NewSigner(prefs, digestPref, getHeadersToSign, httpsig.Signature, 0)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.activitypub.CommonBehavior.NewTransport", "Error creating get signer")
	}

	postSigner, _, err := httpsig.NewSigner(prefs, digestPref, postHeadersToSign, httpsig.Signature, 0)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.activitypub.CommonBehavior.NewTransport", "Error creating post signer")
	}

	privateKey, err := service.encryptionKeyService.GetPrivateKey(userID)

	if err != nil {
		return nil, derp.Wrap(err, "gofed.activitypub.CommonBehavior.NewTransport", "Error loading private key", userID)
	}

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	t = pub.NewHttpSigTransport(
		client,
		"emissary.social",
		NewClock(),
		getSigner,
		postSigner,
		user.ActivityPubPublicKeyURL(),
		privateKey,
	)

	return
}
