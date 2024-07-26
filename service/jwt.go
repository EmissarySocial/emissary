package service

import (
	"crypto/aes"
	"net/http"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/convert"
	"github.com/golang-jwt/jwt/v5"
	"github.com/maypok86/otter"
	"github.com/rs/zerolog/log"
)

// JWT is a service that generates and validates JWT keys.
type JWT struct {
	collection       data.Collection             // Database collection where JWT keys are stored
	cache            otter.Cache[string, []byte] // In-Memory cache for frequently used keys
	hasCache         bool                        // Flag to indicate if the cache is enabled
	keyEncryptingKey []byte                      // "Key Encrypting Key" used to encode/decode JWT keys that are stored in the collection
}

func NewJWT() JWT {
	return JWT{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (service *JWT) Refresh(collection data.Collection, keyEncryptingKey []byte) {
	service.collection = collection
	service.keyEncryptingKey = keyEncryptingKey

	builder := otter.MustBuilder[string, []byte](32).
		WithTTL(24 * time.Hour)

	if cache, err := builder.Build(); err == nil {
		service.cache = cache
		service.hasCache = true
	} else {
		derp.Report(derp.Wrap(err, "service.JWT.Refresh", "Error creating cache"))
		service.hasCache = false
	}
}

func (service *JWT) Close() {
	service.cache.Close()
}

/******************************************
 * sternako.KeyService Methods
 ******************************************/

// NewJWTKey returns a a new JWT Key to the caller.
// This method is a part of the steranko.KeyService interface.
func (service *JWT) NewJWTKey() (string, any, error) {

	const location = "service.JWT.NewJWTKey"

	// New keys are generated for each day
	keyName := time.Now().Format("20060102")

	// If the key exists in the cache or database, then return it
	if result, err := service.load(keyName); err == nil {
		log.Trace().Str("keyName", keyName).Msg("JWT Key Found")
		return keyName, result, nil
	}

	log.Trace().Str("keyName", keyName).Msg("JWT Key NOT FOUND... Looking in Database")

	// If not found, then we will make a new key
	jwtKey, err := service.newJWTKey(keyName)

	if err != nil {
		return "", must(random.GenerateBytes(128)), derp.Wrap(err, location, "Failed Generating Key")
	}

	// Save the new key to the database
	if err := service.save(&jwtKey); err != nil {
		return "", must(random.GenerateBytes(128)), derp.Wrap(err, location, "Failed Saving Key")
	}

	// Return the new key to the caller
	return keyName, jwtKey.Plaintext, nil
}

// FindJWTKey retrieves a key from the cache or database, and returns it to the caller.
// This method is a part of the steranko.KeyService interface.
func (service *JWT) FindJWTKey(token *jwt.Token) (any, error) {

	// Load the key from the cache/database
	keyName := convert.String(token.Header["kid"])
	plaintext, err := service.load(keyName)

	if err != nil {
		return nil, derp.Wrap(err, "service.JWT.FindJWTKey", "Error loading JWT Key", keyName)
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

	claims := model.NewAuthorization()

	result, err := jwt.ParseWithClaims(tokenString, &claims, service.FindJWTKey, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))

	if err != nil {
		return nil, derp.ReportAndReturn(derp.Wrap(err, "service.JWT.Parse", "Error parsing JWT token"))
	}

	return result, nil
}

/******************************************
 * Database Methods
 ******************************************/

// newJWT creates a new plaintext jwt key
func (service *JWT) newJWTKey(keyName string) (model.JWTKey, error) {

	const location = "service.JWT.newJWTKey"

	result := model.NewJWTKey()

	// Generate Key Plaintext
	plaintext, err := random.GenerateBytes(128)

	if err != nil {
		return model.JWTKey{}, derp.Wrap(err, location, "Error generating plaintext")
	}

	// Set the plaintext value of the key
	result.KeyName = keyName
	result.Plaintext = plaintext
	return result, nil
}

// load retrieves a key from the cache or database.  Automatically
// decrypting its plaintext value.  If the key is not found, an
// error is returned.
func (service *JWT) load(keyName string) ([]byte, error) {

	if service.hasCache {
		if plaintext, exists := service.cache.Get(keyName); exists {
			log.Trace().Str("keyName", keyName).Msg("JWT Key Cache Hit")
			return plaintext, nil
		}
	}

	log.Trace().Msg("JWT Key Cache Miss")

	// Try to load the key from the database
	criteria := exp.Equal("keyName", keyName)
	jwtKey := model.NewJWTKey()

	if err := service.collection.Load(criteria, &jwtKey); err != nil {
		return nil, derp.ReportAndReturn(derp.Wrap(err, "service.JWT.load", "Error loading JWT Key"))
	}

	// Decrypt the key in memory
	if err := service.decrypt(&jwtKey); err != nil {
		return nil, derp.ReportAndReturn(derp.Wrap(err, "service.JWT.load", "Error decrypting JWT Key"))
	}

	// Save the plaintext in the memory cache
	if service.hasCache {
		log.Trace().Str("keyName", keyName).Msg("JWT Key Cache Set (load)")
		service.cache.Set(keyName, jwtKey.Plaintext)
	}

	// Return the plaintext to the rest of the application
	return jwtKey.Plaintext, nil
}

// save stores a key in the cache and database, automatically
// encrypting its plaintext value.
func (service *JWT) save(jwtKey *model.JWTKey) error {

	// Encrypt the key in memory
	if err := service.encrypt(jwtKey); err != nil {
		return derp.Wrap(err, "service.JWT.save", "Error encrypting JWT Key")
	}

	// Save the key to the database
	if err := service.collection.Save(jwtKey, ""); err != nil {
		return derp.Wrap(err, "service.JWT.save", "Error saving JWT Key")
	}

	// Apply the item back into the cache
	if service.hasCache {
		log.Trace().Str("keyName", jwtKey.KeyName).Msg("JWT Key Cache Set (save)")
		service.cache.Set(jwtKey.KeyName, jwtKey.Plaintext)
	}

	return nil
}

/******************************************
 * Encryption Methods
 ******************************************/

// encrypt encrypts the plaintext field of the JWTKey
// and stores the result in the encryptedValue field.
func (service *JWT) encrypt(jwtKey *model.JWTKey) error {

	// Create an AES cipher
	cipher, err := aes.NewCipher(service.keyEncryptingKey)

	if err != nil {
		return derp.Wrap(err, "service.JWT.encrypt", "Error creating AES cipher")
	}

	// Encrypt the plaintext
	cipher.Encrypt(jwtKey.EncryptedValue, jwtKey.Plaintext)
	jwtKey.Algorithm = "AES"
	return nil
}

// decrypt decrypts the encryptedValue field of the JWTKey
// and stores the result in the plaintext field.
func (service *JWT) decrypt(jwtKey *model.JWTKey) error {

	// Create an AES cipher
	cipher, err := aes.NewCipher(service.keyEncryptingKey)

	if err != nil {
		return derp.Wrap(err, "service.JWT.decrypt", "Error creating AES cipher")
	}

	// Decrypt the key in memory
	cipher.Decrypt(jwtKey.Plaintext, jwtKey.EncryptedValue)
	return nil
}
