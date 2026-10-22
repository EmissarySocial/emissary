package service

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// mobilePushVAPIDLifetime is how long a signed VAPID token stays valid (push services allow up to 24 hours)
const mobilePushVAPIDLifetime = 12 * time.Hour

// MobilePush delivers push messages to mobile apps. It follows Mastodon's push pattern ("aesgcm"
// encryption, salt and server key in headers), which most mobile clients and their relays implement.
type MobilePush struct {
	webPush *WebPush
}

// NewMobilePush returns a fully initialized MobilePush service
func NewMobilePush() MobilePush {
	return MobilePush{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *MobilePush) Refresh(factory *Factory) {
	service.webPush = factory.WebPush()
}

// Close stops any background processes controlled by this service
func (service *MobilePush) Close() {
	// Nothing to close.
}

/******************************************
 * Delivery
 ******************************************/

// Send delivers one push message to a mobile app's push relay. It returns the relay's status
// code so the caller can prune subscriptions on 404/410, like WebPush.Send.
func (service *MobilePush) Send(session data.Session, endpoint string, p256dh string, auth string, payload []byte) (int, error) {

	const location = "service.MobilePush.Send"

	webPush := service.webPush
	public, private, err := webPush.vapidKeys(session)

	if err != nil {
		return 0, derp.Wrap(err, location, "Loading VAPID keys")
	}

	ciphertext, salt, serverPublic, err := encryptAesgcm(p256dh, auth, payload)

	if err != nil {
		return 0, derp.Wrap(err, location, "Encrypting payload", endpoint)
	}

	subscriber := vapidSubscriber(webPush.ownerEmail, webPush.hostname)
	token, err := signVAPIDToken(endpoint, subscriber, private)

	if err != nil {
		return 0, derp.Wrap(err, location, "Signing VAPID token", endpoint)
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(ciphertext))

	if err != nil {
		return 0, derp.Wrap(err, location, "Creating request", endpoint)
	}

	// Headers as Mastodon sends them: the relays forward Encryption and Crypto-Key to the device
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Content-Encoding", "aesgcm")
	request.Header.Set("Encryption", "salt="+base64.RawURLEncoding.EncodeToString(salt))
	request.Header.Set("Crypto-Key", "dh="+base64.RawURLEncoding.EncodeToString(serverPublic)+";p256ecdsa="+public)
	request.Header.Set("Authorization", "WebPush "+token)
	request.Header.Set("TTL", strconv.Itoa(webPushTTL))

	// Deliver through the guarded client so the request cannot reach a non-public address.
	client := webPush.httpClient
	if client == nil {
		client = webPushHTTPClient(false)
	}

	log.Trace().
		Str("endpoint", endpoint).
		Str("vapidSubscriber", subscriber).
		Msg("MobilePush: sending to push relay")

	response, err := client.Do(request)

	if err != nil {
		return 0, derp.Wrap(err, location, "Sending mobile push", endpoint, subscriber)
	}

	return pushResponseStatus(response, location, endpoint, subscriber), nil
}

// signVAPIDToken returns a VAPID JWT (RFC 8292) for the endpoint's push service, signed with the
// domain's VAPID private key
func signVAPIDToken(endpoint string, subscriber string, privateKey string) (string, error) {

	const location = "service.signVAPIDToken"

	endpointURL, err := url.Parse(endpoint)

	if err != nil {
		return "", derp.Wrap(err, location, "Parsing endpoint", endpoint)
	}

	privateBytes, err := decodeBase64URL(privateKey)

	if err != nil {
		return "", derp.Wrap(err, location, "Decoding VAPID private key")
	}

	signingKey, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), privateBytes)

	if err != nil {
		return "", derp.Wrap(err, location, "Parsing VAPID private key")
	}

	claims := jwt.MapClaims{
		"aud": endpointURL.Scheme + "://" + endpointURL.Host,
		"exp": time.Now().Add(mobilePushVAPIDLifetime).Unix(),
		"sub": "mailto:" + subscriber,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(signingKey)

	if err != nil {
		return "", derp.Wrap(err, location, "Signing token")
	}

	return token, nil
}
