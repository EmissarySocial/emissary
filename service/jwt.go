package service

import (
	"context"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/maypok86/otter"
)

// JWT is a SHARED SERVICE that generates and validates JWT keys.
type JWT struct {
	server    data.Server                 // Server instance for database access
	cache     otter.Cache[string, []byte] // In-Memory cache for frequently used keys
	hasCache  bool                        // Flag to indicate if the cache is enabled
	masterKey string                      // "Key Encrypting Key" used to encode/decode JWT keys that are stored in the collection
}

func NewJWT() JWT {
	return JWT{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *JWT) Refresh(server data.Server, masterKey string) {

	service.server = server
	service.masterKey = masterKey

	builder := otter.MustBuilder[string, []byte](32).
		WithTTL(24 * time.Hour)

	if cache, err := builder.Build(); err == nil {
		service.cache = cache
		service.hasCache = true
	} else {
		derp.Report(derp.Wrap(err, "service.JWT.Refresh", "Unable to create memory cache"))
		service.hasCache = false
	}
}

func (service *JWT) Close() {
	service.cache.Close()
}

/******************************************
 * sternako.KeyService Methods
 ******************************************/

// GetCurrentKey returns a the currently in-use encryption key.
// This method is a part of the steranko.KeyService interface.
func (service *JWT) GetCurrentKey() (string, any, error) {

	const location = "service.JWT.GetCurrentKey"

	// New keys are generated for each day
	keyName := time.Now().Format("20060102")

	// If the key exists in the cache or database, then return it
	if plaintext, err := service.load(keyName); err == nil {
		return keyName, plaintext, nil
	}

	// If not found, then we will make a new key
	plaintext, err := service.create(keyName)

	if err != nil {
		return "", nil, derp.Wrap(err, location, "Unable to create JWT key")
	}

	// Return the new key to the caller
	return keyName, plaintext, nil
}

// FindKey returns the key named in the token.  It uses
// a cache to store frequently used keys, and a database for
// persistent storage.
// This method is a part of the steranko.KeyService interface.
func (service *JWT) FindKey(token *jwt.Token) (any, error) {

	const location = "service.JWT.FindKey"

	// Load the key from the cache/database
	keyName := convert.String(token.Header["kid"])
	plaintext, err := service.load(keyName)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to load JWT Key", keyName)
	}

	// Return the key plaintext
	return plaintext, nil
}

// Parse retrieves a JWT token from the request, and parses it into a JWT token.
// This method is a part of the steranko.KeyService interface.
func (service *JWT) Parse(request *http.Request) (*jwt.Token, error) {
	authorization := request.Header.Get("Authorization")
	authorization = strings.TrimPrefix(authorization, "Bearer ")
	return service.ParseString(authorization)
}

func (service *JWT) ParseString(tokenString string) (*jwt.Token, error) {

	const location = "service.JWT.ParseString"

	// RULE: JWT token must not be empty
	if tokenString == "" {
		return nil, derp.BadRequestError(location, "JWT token cannot be empty")
	}

	// Try to parse the JWT token
	claims := model.NewAuthorization()
	result, err := jwt.ParseWithClaims(tokenString, &claims, service.FindKey, steranko.JWTValidMethods())

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to parse JSON Web Token", tokenString)
	}

	// Success.
	return result, nil
}

/******************************************
 * Database Methods
 ******************************************/

func (service *JWT) collection(ctx context.Context) (data.Collection, error) {

	const location = "service.JWT.collection"

	session, err := service.server.Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to connect to database")
	}

	if session == nil {
		return nil, derp.InternalError(location, "Database session is nil. This should never happen.")
	}

	return session.Collection("JWT"), nil
}

// create creates a new plaintext jwt key
func (service *JWT) create(keyName string) ([]byte, error) {

	const location = "service.JWT.create"

	// Generate Key Plaintext
	plaintext, err := random.GenerateBytes(128)

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to generate random bytes")
	}

	// Get encrypted value of the key
	encrypted, err := service.encrypt(plaintext)

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to encrypt JWT key")
	}

	// Set the plaintext value of the key
	record := model.NewJWTKey()
	record.Algorithm = "PLAINTEXT"
	record.KeyName = keyName
	record.Encrypted = hex.EncodeToString(encrypted)

	// Apply the item back into the cache
	if service.hasCache {
		service.cache.Set(keyName, plaintext)
	}

	// Create a request context that times out after 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	// Save the key to the database
	collection, err := service.collection(ctx)

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to connect to JWT collection")
	}

	if collection == nil {
		return []byte{}, derp.InternalError(location, "JWT collection is nil. This should never happen.")
	}

	if err := collection.Save(&record, "New key created"); err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to save JWT key")
	}

	// Return the plaintext value of the key
	return plaintext, nil
}

// load retrieves a key from the cache or database.  Automatically
// decrypting its plaintext value.  If the key is not found, an
// error is returned.
func (service *JWT) load(keyName string) ([]byte, error) {

	const location = "service.JWT.load"

	// If the key is in the cache, then return it
	if service.hasCache {
		if plaintext, exists := service.cache.Get(keyName); exists {
			return plaintext, nil
		}
	}

	// Create a request context that times out after 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	// Try to load the key from the database
	collection, err := service.collection(ctx)

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to connect to JWT collection")
	}

	criteria := exp.Equal("keyName", keyName)
	jwtKey := model.NewJWTKey()

	if err := collection.Load(criteria, &jwtKey); err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to load JWT key")
	}

	// Decode Base64 text into a slice of bytes
	encrypted, err := hex.DecodeString(jwtKey.Encrypted)

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to decode base64 key")
	}

	// Decrypt the encrypted value into a usable plaintext
	plaintext, err := service.decrypt(encrypted)

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to decrypt JWT key")
	}

	// Save the plaintext in the memory cache
	if service.hasCache {
		service.cache.Set(keyName, plaintext)
	}

	// Return the plaintext to the rest of the application
	return plaintext, nil
}

/******************************************
 * Encryption Methods
 ******************************************/

func (service *JWT) NewToken(claims jwt.Claims) (string, error) {

	const location = "service.JWT.NewToken"

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	// Get the signing key from the KeyService
	keyID, key, err := service.GetCurrentKey()

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to retrieve JWT key")
	}

	token.Header["kid"] = keyID

	// Try to generate encoded token
	result, err := token.SignedString(key)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to sign JWT token")
	}

	// Return the encoded JWT
	return result, nil
}

func (service *JWT) ParseToken(tokenString string, claims jwt.Claims) error {

	const location = "service.JWT.ParseToken"

	// Try to parse the JWT token using this key service
	if _, err := jwt.ParseWithClaims(tokenString, claims, service.FindKey, jwt.WithValidMethods([]string{"HS512"})); err != nil {
		return derp.Wrap(err, location, "Unable to parse JSON Web Token", tokenString)
	}

	// You're so beautiful.
	return nil
}

/******************************************
 * Encryption Methods
 ******************************************/

// encrypt uses the service's KEK to encrypt the plaintext into an encrypted value.
func (service *JWT) encrypt(plaintext []byte) ([]byte, error) {

	return plaintext, nil

	// TODO: Restore Key-Encrypting-Key feature for JWT keys.
	// The following commented code does not work because the AES algorithm
	// only works with fixed-size blocks, so encrypted data was being truncated
	// at the first block boundary. Instead,  need to use a GCM mode as described in:
	// https://stackoverflow.com/questions/75064248/golang-aes-decryption-is-not-returning-same-text
	/*
		const location = "service.JWT.encrypt"

		// Create an AES cipher
		cipher, err := aes.NewCipher(service.keyEncryptingKey)

		if err != nil {
			return []byte{}, derp.Wrap(err, location, "Error creating AES cipher")
		}

		// Encrypt the plaintext
		result := make([]byte, 128)
		cipher.Encrypt(result, plaintext)

		return result, nil
	*/
}

// decrypt uses the service's KEK to decrypt an encrypted value into plaintext
func (service *JWT) decrypt(encrypted []byte) ([]byte, error) {

	return encrypted, nil
	/*
		const location = "service.JWT.decrypt"

		// Create an AES cipher
		cipher, err := aes.NewCipher(service.keyEncryptingKey)

		if err != nil {
			return []byte{}, derp.Wrap(err, location, "Error creating AES cipher")
		}

		// Decrypt the key in memory
		result := make([]byte, 128)
		cipher.Decrypt(result, encrypted)

		return result, nil
	*/
}
