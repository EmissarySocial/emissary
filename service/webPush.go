package service

import (
	"io"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"syscall"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/uri"
	"github.com/rs/zerolog/log"
)

// webPushErrorBodyMaxLength caps how much of a push service's error response is read back, so a
// hostile or broken service cannot stream unbounded data into the log
const webPushErrorBodyMaxLength = 1024

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
	ownerEmail      string
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

	// The Domain owner is this server's real human contact -- the same address DomainEmail uses as
	// its return address -- which is exactly what the VAPID "sub" claim is for (see vapidSubscriber).
	service.ownerEmail = factory.config.Owner.EmailAddress

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
		return "", "", derp.Wrap(err, location, "Generating VAPID keys")
	}

	updated := *domain
	if updated.Data == nil {
		return "", "", derp.Internal(location, "Domain.Data map is not initialized")
	}
	updated.Data[domainDataVAPIDPublicKey] = newPublic
	updated.Data[domainDataVAPIDPrivateKey] = newPrivate

	if err := service.domainService.Save(session, updated, "Generate VAPID keys"); err != nil {
		return "", "", derp.Wrap(err, location, "Saving VAPID keys")
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
		return 0, derp.Wrap(err, location, "Loading VAPID keys")
	}

	subscription := &webpush.Subscription{
		Endpoint: endpoint,
		Keys: webpush.Keys{
			P256dh: p256dh,
			Auth:   auth,
		},
	}

	// Deliver through the guarded client so the request cannot reach a non-public address.
	// Fall back to a guarded client if the service was never refreshed (fail closed, not open).
	client := service.httpClient
	if client == nil {
		client = webPushHTTPClient(false)
	}

	// NOTE: this is BARE ("admin@example.com"), and must stay that way -- webpush-go prepends the
	// "mailto:" scheme itself.  See vapidSubscriber.
	subscriber := vapidSubscriber(service.ownerEmail, service.hostname)

	log.Trace().
		Str("endpoint", endpoint).
		Str("vapidSubscriber", subscriber).
		Msg("WebPush: sending to push service")

	response, err := webpush.SendNotification(payload, subscription, &webpush.Options{
		HTTPClient:      client,
		Subscriber:      subscriber,
		VAPIDPublicKey:  public,
		VAPIDPrivateKey: private,
		TTL:             webPushTTL,
	})

	if err != nil {
		return 0, derp.Wrap(err, location, "Sending Web Push notification", endpoint, subscriber)
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			derp.Report(derp.Wrap(err, location, "Closing response body", endpoint, subscriber))
		}
	}()

	// A push service states WHY it rejected a message in the response body -- Apple returns
	// `{"reason":"BadJwtToken"}`, for instance -- and the status code alone is rarely enough to act
	// on.  The subscriber rides along because a malformed "sub" claim is a common cause.
	if (response.StatusCode < 200) || (response.StatusCode > 299) {

		// Best-effort: a body we cannot read must not mask the status code we CAN report.
		body, _ := io.ReadAll(io.LimitReader(response.Body, webPushErrorBodyMaxLength))

		log.Warn().
			Int("statusCode", response.StatusCode).
			Str("endpoint", endpoint).
			Str("vapidSubscriber", subscriber).
			Str("response", string(body)).
			Msg("WebPush: push service REJECTED this message")
	}

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
 * VAPID Subscriber
 ******************************************/

// vapidSubscriberFallback is the contact address used when no valid one can be derived
const vapidSubscriberFallback = "admin@example.com"

// vapidSubscriber returns the VAPID "sub" claim (RFC 8292): a contact address for whoever operates
// this server
func vapidSubscriber(adminEmail string, hostname string) string {

	// RULE: every address returned here MUST be bare.  webpush-go prepends "mailto:" to any
	// subscriber that does not start with "https:", so a "mailto:" URI double-prefixes inside the
	// signed token.  Push services reject the whole JWT for that, naming neither the claim nor the
	// scheme: Apple answers `{"reason":"BadJwtToken"}` with an HTTP 403.

	// The configured administrator IS the contact this claim is meant to carry, so prefer it.
	if isBareEmailAddress(adminEmail) {
		return adminEmail
	}

	// Otherwise derive one from the hostname.  Reduce any shape (bare host, "host:port", a full
	// URL) to its bare hostname, so a port cannot survive into the address.
	host := uri.Hostname(hostname)

	// A local/dev host cannot yield a contact anyone could reach.
	if uri.IsLocalHostname(host) {
		return vapidSubscriberFallback
	}

	// An email domain needs at least one dot, so a single-label host cannot be one.
	if !strings.Contains(host, ".") {
		return vapidSubscriberFallback
	}

	return "admin@" + host
}

// isBareEmailAddress reports whether the value is a valid email address and nothing else -- no
// display name ("Ben <ben@example.com>"), no scheme
func isBareEmailAddress(value string) bool {

	if value == "" {
		return false
	}

	address, err := mail.ParseAddress(value)

	if err != nil {
		return false
	}

	return address.Address == value
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
