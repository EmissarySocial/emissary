package service

import (
	"net"
	"net/http"
	"syscall"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// Domain.Data keys under which the per-domain VAPID keypair is stored (generated lazily).
const domainDataVAPIDPublicKey = "vapidPublicKey"
const domainDataVAPIDPrivateKey = "vapidPrivateKey"

// webPushTTL is the number of seconds a push message may be queued by the push service.
const webPushTTL = 3600

// webPushTimeout caps how long a single push-delivery HTTP request may run.
const webPushTimeout = 30 * time.Second

// WebPush encapsulates per-domain VAPID key management and Web Push delivery (RFC 8291).  Emissary
// is multi-tenant, so each domain has its own VAPID keypair, generated lazily on first use.
type WebPush struct {
	domainService   *Domain
	hostname        string
	allowPrivateIPs bool
	httpClient      *http.Client
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

	// A local/dev instance (served from a local hostname) may reach private addresses so it can
	// talk to itself; a production instance must not.  This mirrors ActivityStream.AllowPrivateIPs()
	// so push delivery and ActivityPub delivery apply the same network policy.
	service.allowPrivateIPs = uri.IsLocalHostname(service.hostname)
	service.httpClient = webPushHTTPClient(service.allowPrivateIPs)
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

	// Deliver through the guarded client so the request cannot reach a non-public address.
	// Fall back to a guarded client if the service was never refreshed (fail closed, not open).
	client := service.httpClient
	if client == nil {
		client = webPushHTTPClient(false)
	}

	response, err := webpush.SendNotification(payload, subscription, &webpush.Options{
		HTTPClient:      client,
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

// EndpointIsAllowed reports whether a Web Push endpoint may be registered or delivered to.  A
// production instance rejects endpoints whose host is a loopback/private/link-local/metadata name
// or IP literal; a local/dev instance allows them.  This is a fast, best-effort check on the literal
// URL — the connection-time dialer guard in Send is the authoritative, DNS-rebinding-safe backstop.
func (service *WebPush) EndpointIsAllowed(endpoint string) bool {

	if service.allowPrivateIPs {
		return true
	}

	return uri.NotLocalURL(endpoint)
}

/******************************************
 * SSRF-Hardened HTTP Client
 ******************************************/

// webPushHTTPClient returns the HTTP client used to deliver Web Push messages.  Unless the instance
// is permitted to reach private addresses, the client's dialer refuses to connect to any non-public
// (loopback/private/link-local/cloud-metadata) IP.  The address is validated at connection time —
// after DNS resolution — so a subscription endpoint cannot be re-pointed at an internal host via
// DNS rebinding.  (SSRF, CWE-918.)
func webPushHTTPClient(allowPrivateIPs bool) *http.Client {

	if allowPrivateIPs {
		return &http.Client{Timeout: webPushTimeout}
	}

	dialer := &net.Dialer{
		Timeout:   webPushTimeout,
		KeepAlive: 30 * time.Second,
		Control:   blockNonPublicAddress,
	}

	// Clone the standard transport so we keep its pooling/proxy/TLS defaults, then swap in the
	// guarded dialer.  Fall back to a bare transport if the default is ever not an *http.Transport.
	var transport *http.Transport
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = base.Clone()
	} else {
		transport = &http.Transport{}
	}
	transport.DialContext = dialer.DialContext

	return &http.Client{
		Timeout:   webPushTimeout,
		Transport: transport,
	}
}

// blockNonPublicAddress is a net.Dialer.Control hook that rejects a connection whose resolved
// address is not a public IP.  Control runs after DNS resolution with the concrete "ip:port" the
// socket is about to connect to, which is what makes this guard safe against DNS rebinding.
func blockNonPublicAddress(_ string, address string, _ syscall.RawConn) error {

	const location = "service.blockNonPublicAddress"

	host, _, err := net.SplitHostPort(address)

	if err != nil {
		return derp.Wrap(err, location, "Invalid dial address", address)
	}

	if !uri.IsPublicIPAddress(host) {
		return derp.Forbidden(location, "Blocked Web Push delivery to non-public address", address)
	}

	// This IP is cleared for takeoff.
	return nil
}
