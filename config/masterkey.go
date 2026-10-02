package config

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/benpate/derp"
	"github.com/rs/zerolog/log"
)

// masterKeyLength is the required length of a raw (decoded) master key: 32 bytes, for AES-256.
const masterKeyLength = 32

// NewMasterKey returns a new, random master key as a 64-character hexadecimal string.
func NewMasterKey() string {

	// Generate 32 cryptographically random bytes.  crypto/rand.Read never returns
	// an error (it crashes the program on catastrophic failure), so it is safe to ignore.
	masterKey := make([]byte, masterKeyLength)
	_, _ = rand.Reader.Read(masterKey)

	return hex.EncodeToString(masterKey)
}

// DecodeMasterKey converts a domain's hex-encoded master key into the raw 32-byte AES-256 key.
func DecodeMasterKey(masterKey string) ([]byte, error) {

	const location = "config.DecodeMasterKey"

	// RULE: An empty master key is a configuration error, not a valid key.
	// hex.DecodeString("") succeeds with a zero-length slice, which would otherwise
	// surface much later as an opaque AES cipher-size error (BUG-110).
	if masterKey == "" {
		return nil, derp.Internal(location, "This domain has no masterKey configured. Add a 64-character hexadecimal 'masterKey' to this domain's block in the server configuration, then restart Emissary.")
	}

	// Decode the hexadecimal string into raw bytes
	result, err := hex.DecodeString(masterKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "The configured masterKey is not valid hexadecimal")
	}

	// RULE: The decoded key must be exactly 32 bytes (AES-256)
	if len(result) != masterKeyLength {
		return nil, derp.Internal(location, "The configured masterKey must be exactly 64 hexadecimal characters (32 bytes)")
	}

	// The key to my heart is 32 bytes long.
	return result, nil
}

// ReportInvalidMasterKeys logs an actionable error for every domain in this Config
// whose masterKey is missing or malformed.
func (config Config) ReportInvalidMasterKeys() {

	// A broken domain still boots -- so the operator can reach the rest of the system --
	// but Connections and Merchant Accounts cannot be saved until the key is fixed,
	// so make the cause and the remedy impossible to miss in the log.
	for _, domain := range config.Domains {

		if _, err := DecodeMasterKey(domain.MasterKey); err != nil {
			log.Error().Str("hostname", domain.Hostname).Msg("This domain has a missing or invalid masterKey. External Connections and Merchant Accounts cannot be saved until a 64-character hexadecimal 'masterKey' is set in this domain's block in the server configuration.")
		}
	}
}
