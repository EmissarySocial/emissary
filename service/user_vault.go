package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

/******************************************
 * User Vault
 *
 * A User's Vault holds the secrets for their external service connections,
 * sealed with the domain's master key -- the same key that protects the
 * Connection and MerchantAccount vaults.
 ******************************************/

// encryptVault seals any plaintext values written into this User's Vault
func (service *User) encryptVault(user *model.User) error {

	const location = "service.User.encryptVault"

	// RULE: do not reach for the master key unless there is something to seal. A domain
	// with a missing or malformed masterKey exists in the wild, and User.Save is on the
	// hot path -- decoding unconditionally would stop every User on it from saving.
	if !user.Vault.NeedsEncryption() {
		return nil
	}

	// Decode the domain's master key
	encryptionKey, err := config.DecodeMasterKey(service.masterKey)

	if err != nil {
		return derp.Wrap(err, location, "Decoding encryption key")
	}

	// Seal the Vault
	if err := user.Vault.Encrypt(encryptionKey); err != nil {
		return derp.Wrap(err, location, "Encrypting vault values")
	}

	// The secrets are sealed. Nobody gets in without the key.
	return nil
}

// DecryptVault opens this User's Vault and returns the named secrets in plaintext,
// or every value it holds when no names are given
func (service *User) DecryptVault(user *model.User, values ...string) (mapof.String, error) {

	const location = "service.User.DecryptVault"

	// The result is secret material: never log it, never put it in an error detail,
	// and never write it back onto a model object.

	// NILCHECK: user cannot be nil
	if user == nil {
		return nil, derp.Internal(location, "User cannot be nil")
	}

	// Decode the domain's master key
	encryptionKey, err := config.DecodeMasterKey(service.masterKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Decoding encryption key")
	}

	// Open the Vault
	result, err := user.Vault.Decrypt(encryptionKey, values...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Decrypting vault")
	}

	// Open sesame.
	return result, nil
}
