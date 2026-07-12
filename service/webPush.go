package service

import (
	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/benpate/data"
	"github.com/benpate/derp"
)

// Domain.Data keys under which the per-domain VAPID keypair is stored (generated lazily).
const domainDataVAPIDPublicKey = "vapidPublicKey"
const domainDataVAPIDPrivateKey = "vapidPrivateKey"

// webPushTTL is the number of seconds a push message may be queued by the push service.
const webPushTTL = 3600

// WebPush encapsulates per-domain VAPID key management and Web Push delivery (RFC 8291).  Emissary
// is multi-tenant, so each domain has its own VAPID keypair, generated lazily on first use.
type WebPush struct {
	domainService *Domain
	hostname      string
}

// NewWebPush returns a fully initialized WebPush service
func NewWebPush() WebPush {
	return WebPush{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *WebPush) Refresh(factory *Factory) {
	service.domainService = factory.Domain()
	service.hostname = factory.Hostname()
}

// Close stops any background processes controlled by this service
func (service *WebPush) Close() {
	// Nothin to do here.
}

/******************************************
 * VAPID Keys
 ******************************************/

// PublicKey returns the domain's VAPID public key (generating + persisting the keypair on first use).
// Templates expose this to the browser so it can subscribe.
func (service *WebPush) PublicKey(session data.Session) (string, error) {
	public, _, err := service.vapidKeys(session)
	return public, err
}

// vapidKeys returns the domain's VAPID keypair, lazily generating and persisting it on first use.
func (service *WebPush) vapidKeys(session data.Session) (publicKey string, privateKey string, err error) {

	const location = "service.WebPush.vapidKeys"

	domain := service.domainService.Get()

	public := domain.Data.GetString(domainDataVAPIDPublicKey)
	private := domain.Data.GetString(domainDataVAPIDPrivateKey)

	if public != "" && private != "" {
		return public, private, nil
	}

	// Generate a fresh keypair and persist it on the Domain record.
	newPrivate, newPublic, err := webpush.GenerateVAPIDKeys()

	if err != nil {
		return "", "", derp.Wrap(err, location, "Unable to generate VAPID keys")
	}

	updated := *domain
	if updated.Data == nil {
		return "", "", derp.Internal(location, "Domain.Data map is not initialized")
	}
	updated.Data[domainDataVAPIDPublicKey] = newPublic
	updated.Data[domainDataVAPIDPrivateKey] = newPrivate

	if err := service.domainService.Save(session, updated, "Generate VAPID keys"); err != nil {
		return "", "", derp.Wrap(err, location, "Unable to save VAPID keys")
	}

	return newPublic, newPrivate, nil
}

/******************************************
 * Delivery
 ******************************************/

// Send delivers a single push message to one subscription.  It returns the HTTP status code from the
// push service so the caller can prune subscriptions on 404/410.  A non-2xx status is not treated as
// a Go error here (the caller decides what to do); a transport error IS returned.
func (service *WebPush) Send(session data.Session, endpoint string, p256dh string, auth string, payload []byte) (int, error) {

	const location = "service.WebPush.Send"

	public, private, err := service.vapidKeys(session)

	if err != nil {
		return 0, derp.Wrap(err, location, "Unable to load VAPID keys")
	}

	subscription := &webpush.Subscription{
		Endpoint: endpoint,
		Keys: webpush.Keys{
			P256dh: p256dh,
			Auth:   auth,
		},
	}

	subscriber := "admin@" + service.hostname

	response, err := webpush.SendNotification(payload, subscription, &webpush.Options{
		Subscriber:      "mailto:" + subscriber,
		VAPIDPublicKey:  public,
		VAPIDPrivateKey: private,
		TTL:             webPushTTL,
	})

	if err != nil {
		return 0, derp.Wrap(err, location, "Unable to send Web Push notification", endpoint)
	}

	defer response.Body.Close()

	return response.StatusCode, nil
}
