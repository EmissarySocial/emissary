package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// testPushClient is a mobile app's push keys: the private key stays on the device
type testPushClient struct {
	privateKey *ecdh.PrivateKey
	auth       []byte
}

// newTestPushClient generates push keys the way a mobile app does
func newTestPushClient(t *testing.T) testPushClient {
	t.Helper()

	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)

	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)

	return testPushClient{privateKey: privateKey, auth: auth}
}

// p256dh returns the client's public key as clients send it
func (client testPushClient) p256dh() string {
	return base64.RawURLEncoding.EncodeToString(client.privateKey.PublicKey().Bytes())
}

// authSecret returns the client's auth secret as clients send it
func (client testPushClient) authSecret() string {
	return base64.RawURLEncoding.EncodeToString(client.auth)
}

// decrypt reverses aesgcm exactly as Ice Cubes and the official app do (NotificationServiceSupport.swift),
// written out separately so it does not share code with the sender
func (client testPushClient) decrypt(t *testing.T, ciphertext []byte, salt []byte, serverPublic []byte) []byte {
	t.Helper()

	serverKey, err := ecdh.P256().NewPublicKey(serverPublic)
	require.NoError(t, err)

	shared, err := client.privateKey.ECDH(serverKey)
	require.NoError(t, err)

	keyMaterial, err := hkdf.Key(sha256.New, shared, client.auth, "Content-Encoding: auth\x00", 32)
	require.NoError(t, err)

	info := func(contentType string) string {
		clientPublic := client.privateKey.PublicKey().Bytes()
		var result []byte
		result = append(result, "Content-Encoding: "...)
		result = append(result, contentType...)
		result = append(result, 0)
		result = append(result, "P-256"...)
		result = append(result, 0, 0, 65)
		result = append(result, clientPublic...)
		result = append(result, 0, 65)
		result = append(result, serverPublic...)
		return string(result)
	}

	key, err := hkdf.Key(sha256.New, keyMaterial, salt, info("aesgcm"), 16)
	require.NoError(t, err)

	nonce, err := hkdf.Key(sha256.New, keyMaterial, salt, info("nonce"), 12)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	require.NoError(t, err)

	// The first two bytes give the padding length
	paddingLength := int(plaintext[0])*256 + int(plaintext[1])
	return plaintext[2+paddingLength:]
}

// TestEncryptAesgcm_ClientCanDecrypt confirms a mobile app recovers the exact payload
func TestEncryptAesgcm_ClientCanDecrypt(t *testing.T) {

	client := newTestPushClient(t)
	payload := []byte(`{"notification_id":1,"title":"Tester mentioned you","body":"hello"}`)

	ciphertext, salt, serverPublic, err := encryptAesgcm(client.p256dh(), client.authSecret(), payload)
	require.NoError(t, err)
	require.Len(t, salt, 16)
	require.Len(t, serverPublic, 65)

	require.Equal(t, payload, client.decrypt(t, ciphertext, salt, serverPublic))
}

// TestEncryptAesgcm_AcceptsPaddedKeys confirms keys sent with base64 padding still work
func TestEncryptAesgcm_AcceptsPaddedKeys(t *testing.T) {

	client := newTestPushClient(t)
	payload := []byte("padded")

	padded := base64.URLEncoding.EncodeToString(client.privateKey.PublicKey().Bytes())
	paddedAuth := base64.URLEncoding.EncodeToString(client.auth)
	require.True(t, strings.HasSuffix(paddedAuth, "="))

	ciphertext, salt, serverPublic, err := encryptAesgcm(padded, paddedAuth, payload)
	require.NoError(t, err)
	require.Equal(t, payload, client.decrypt(t, ciphertext, salt, serverPublic))
}

// TestEncryptAesgcm_RejectsBadKeys confirms an unusable client key is an error, not a garbled push
func TestEncryptAesgcm_RejectsBadKeys(t *testing.T) {

	client := newTestPushClient(t)

	_, _, _, err := encryptAesgcm("not-a-key", client.authSecret(), []byte("x"))
	require.Error(t, err)

	_, _, _, err = encryptAesgcm(client.p256dh(), "%%%", []byte("x"))
	require.Error(t, err)
}

// TestMobilePush_Send delivers to a stand-in push relay and checks what a relay forwards to the
// device: the aesgcm headers, a valid VAPID token, and a body the client decrypts to the payload
func TestMobilePush_Send(t *testing.T) {

	var received *http.Request
	var receivedBody []byte

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer relay.Close()

	domainService, session := newTestDomainService(t, "example.com")
	domainService.publish(storeTestDomain(t, session, "example.com"))
	webPush := WebPush{domainService: domainService, httpClient: webPushHTTPClient(true), hostname: "example.com"}
	service := MobilePush{webPush: &webPush}

	public, _, err := webPush.vapidKeys(session)
	require.NoError(t, err)

	client := newTestPushClient(t)
	payload := []byte(`{"notification_id":1792610942703,"title":"Tester mentioned you","body":"hello"}`)

	status, err := service.Send(session, relay.URL+"/push/device-token/demo@example.com", client.p256dh(), client.authSecret(), payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, received)

	require.Equal(t, "aesgcm", received.Header.Get("Content-Encoding"))
	require.Equal(t, "3600", received.Header.Get("TTL"))

	// The salt and server key travel in headers
	salt, err := decodeBase64URL(strings.TrimPrefix(received.Header.Get("Encryption"), "salt="))
	require.NoError(t, err)

	cryptoKey := received.Header.Get("Crypto-Key")
	dh, vapidPublic, found := strings.Cut(cryptoKey, ";p256ecdsa=")
	require.True(t, found, cryptoKey)
	require.Equal(t, public, vapidPublic)

	serverPublic, err := decodeBase64URL(strings.TrimPrefix(dh, "dh="))
	require.NoError(t, err)

	require.Equal(t, payload, client.decrypt(t, receivedBody, salt, serverPublic))

	// The VAPID token is signed by the domain's key and names the relay as its audience
	token := strings.TrimPrefix(received.Header.Get("Authorization"), "WebPush ")
	vapidKeyBytes, err := decodeBase64URL(public)
	require.NoError(t, err)

	verifyKey, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), vapidKeyBytes)
	require.NoError(t, err)

	parsed, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return verifyKey, nil }, jwt.WithValidMethods([]string{"ES256"}))
	require.NoError(t, err)

	claims := parsed.Claims.(jwt.MapClaims)
	require.Equal(t, relay.URL, claims["aud"])
	require.True(t, strings.HasPrefix(claims["sub"].(string), "mailto:"))
}
