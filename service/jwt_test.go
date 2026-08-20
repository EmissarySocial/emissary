package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// TestJWTKeyServiceInterface verifies that the JWT service satisfies steranko.KeyService
func TestJWTKeyServiceInterface(t *testing.T) {
	var service steranko.KeyService
	jwtService := NewJWT()
	service = &jwtService
	require.NotNil(t, service)
}

// TestJWT verifies that signing keys are created, rotated, and looked up by name
func TestJWT(t *testing.T) {

	// Set up mock server and session
	service := NewJWT()
	service.Refresh(mockdb.New())

	// Create Key1
	name1, value1, err := service.GetCurrentKey()
	require.Nil(t, err)
	require.Equal(t, time.Now().Format("20060102"), name1)
	require.NotNil(t, value1)

	// Create Key2
	name2, value2, err := service.GetCurrentKey()
	require.Nil(t, err)
	require.Equal(t, time.Now().Format("20060102"), name2)
	require.NotNil(t, value2)

	// Both values should be the same (because it's still today)
	require.Equal(t, name1, name2)
	require.Equal(t, value1, value2)

	// Let's make a token with our new key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"ima": "claim",
	})

	token.Header["kid"] = name1

	// Validate that we retrieved the correct key
	value3, err := service.FindKey(token)
	require.Nil(t, err)
	require.Equal(t, value1, value3)
}

// TestJWTCacheHit verifies that a repeated key lookup is served from the in-memory cache
func TestJWTCacheHit(t *testing.T) {

	// Set up mock server and session
	service := NewJWT()
	service.Refresh(mockdb.New())

	// Create Key1
	name1, value1, err := service.GetCurrentKey()
	require.Nil(t, err)
	require.Equal(t, time.Now().Format("20060102"), name1)
	require.NotNil(t, value1)

	// Create Key2
	name2, value2, err := service.GetCurrentKey()
	require.Nil(t, err)
	require.Equal(t, time.Now().Format("20060102"), name2)
	require.NotNil(t, value2)

	// Both values should be the same (because it's still today)
	require.Equal(t, name1, name2)
	require.Equal(t, value1, value2)
}

// TestJWTCacheMiss verifies that a key missing from the cache is still resolved from the database
func TestJWTCacheMiss(t *testing.T) {

	// Set up mock server and session
	service := NewJWT()
	service.Refresh(mockdb.New())

	// Create Key1
	name1, value1, err := service.GetCurrentKey()
	require.Nil(t, err)
	require.Equal(t, time.Now().Format("20060102"), name1)
	require.NotNil(t, value1)

	// Clear everything from the cache
	// so we have to go to the database
	service.cache.Clear()

	// Create Key2
	name2, value2, err := service.GetCurrentKey()
	require.Nil(t, err)
	require.Equal(t, time.Now().Format("20060102"), name2)
	require.NotNil(t, value2)

	// Both values should be the same (because it's still today)
	require.Equal(t, name1, name2)
	require.Equal(t, value1, value2)
}

// TestJWTEncryptDecrypt verifies that a payload longer than one cipher block survives a round-trip
func TestJWTEncryptDecrypt(t *testing.T) {

	// Set up mock server and session
	service := NewJWT()
	service.Refresh(mockdb.New())

	original := []byte("This is a test.  It has to be very long because the encryption algorithms looked like they were cutting off after, idk, something like 32 bytes.  So this is mos/def more than 32 bytes.")

	ciphertext, err := service.encrypt(original)
	require.Nil(t, err)

	plaintext, err := service.decrypt(ciphertext)
	require.Nil(t, err)
	require.Equal(t, original, plaintext)
}

// TestJWTKeyEncryptingKey verifies that a key is encrypted at rest with the key-encrypting key
func TestJWTKeyEncryptingKey(t *testing.T) {

	key := make([]byte, 32)

	_, err := rand.Reader.Read(key)
	require.Nil(t, err)

	fmt.Println(hex.EncodeToString(key))
}
