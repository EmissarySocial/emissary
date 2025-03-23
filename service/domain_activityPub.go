package service

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/sigs"
)

/******************************************
 * Domain/Actor Methods
 ******************************************/

// Hostname returns the domain-only name (no protocol)
func (service *Domain) Hostname() string {
	return service.hostname
}

// ActorID returns the URL for this domain/actor
func (service *Domain) ActorID() string {
	return domain.AddProtocol(service.hostname) + "/@application"
}

// PublicKeyID returns the URL for the public key for this domain/actor
func (service *Domain) PublicKeyID() string {
	return service.ActorID() + "#main-key"
}

// PublicKeyPEM returns the PEM-encoded public key for this domain/actor
func (service *Domain) PublicKeyPEM() (string, error) {

	// Try to retrieve the private key for this domain
	privateKey, err := service.PrivateKey()

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.PublicKeyPEM", "Error getting public key")
	}

	// Encode the public key portion
	publicKeyPEM := sigs.EncodePublicPEM(privateKey)
	return publicKeyPEM, nil
}

// PrivateKey returns the private key for this domain/actor
func (service *Domain) PrivateKey() (*rsa.PrivateKey, error) {

	const location = "service.Domain.PrivateKey"

	// Get the Domain record
	domain, err := service.LoadDomain()

	if err != nil {
		return nil, derp.Wrap(err, location, "Error loading Domain record")
	}

	// Try to use the existing private key
	if domain.PrivateKey != "" {

		privateKey, err := sigs.DecodePrivatePEM(domain.PrivateKey)

		if rsaKey, ok := privateKey.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}

		// Fall through means that we have a value for "domain.PrivateKey" but it's not
		// valid.  So, let's log the error and try to make a new one.
		derp.Report(derp.Wrap(err, location, "Error decoding private key. Creating a new key"))
	}

	// Otherwise, create a new private key, save it, and return it to the caller.
	privateKey, err := rsa.GenerateKey(rand.Reader, encryptionKeyBits)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error generating RSA key")
	}

	// Save the new private key into the Domain record
	domain.PrivateKey = sigs.EncodePrivatePEM(privateKey)

	if err := service.Save(domain, "Generated Private Key"); err != nil {
		return nil, derp.Wrap(err, location, "Error saving new EncryptionKey")
	}

	// Success??
	return privateKey, nil
}

// ActivityPubActor returns an ActivityPub Actor object
// ** WHICH INCLUDES ENCRYPTION KEYS ** for the provided User.
func (service *Domain) ActivityPubActor() (outbox.Actor, error) {

	const location = "service.Domain.ActivityPubActor"

	// Retrieve the Private Key from the Domain record
	privateKey, err := service.PrivateKey()

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error extracting private key")
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(service.ActorID(), privateKey, outbox.WithClient(service.activityStream))

	return actor, nil
}

func (service *Domain) WebFinger() digit.Resource {

	// Make a WebFinger resource for this Stream.
	result := digit.NewResource(service.ActorID()).
		Alias("acct:application@"+service.Hostname()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, service.ActorID()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, service.ActorID())

	return result
}
