package service

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	dt "github.com/benpate/domain"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

/******************************************
 * Domain/Actor Methods
 ******************************************/

// Hostname returns the domain-only name (no protocol)
func (service *Domain) Hostname() string {
	return service.hostname
}

// Host returns the host (with protocol)
func (service *Domain) Host() string {
	return dt.AddProtocol(service.hostname)
}

func (service *Domain) GetJSONLD(session data.Session) (mapof.Any, error) {

	const location = "service.Domain.GetJSONLD"

	// Load the public key PEM
	publicKeyPEM, err := service.PublicKeyPEM(session)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to load public key PEM")
	}

	actorID := service.ActorID()

	domain := service.Get()

	// Return the result as a JSON-LD document
	result := map[string]any{
		vocab.AtContext:                 []any{vocab.ContextTypeActivityStreams, vocab.ContextTypeSecurity, vocab.ContextTypeToot},
		vocab.PropertyType:              vocab.ActorTypeApplication,
		vocab.PropertyID:                actorID,
		vocab.PropertyPreferredUsername: "application",
		vocab.PropertyName:              service.Hostname(),
		vocab.PropertyIcon:              domain.IconURL(),
		vocab.PropertyImage:             domain.IconURL(),
		vocab.PropertyFollowing:         actorID + "/following",
		vocab.PropertyFollowers:         actorID + "/followers",
		vocab.PropertyLiked:             actorID + "/liked",
		vocab.PropertyOutbox:            actorID + "/outbox",
		vocab.PropertyInbox:             actorID + "/inbox",
		vocab.PropertyTootDiscoverable:  false,
		vocab.PropertyTootIndexable:     false,

		vocab.PropertyPublicKey: mapof.Any{
			vocab.PropertyID:           service.PublicKeyID(),
			vocab.PropertyOwner:        actorID,
			vocab.PropertyPublicKeyPEM: publicKeyPEM,
		},
	}

	return result, nil
}

// ActorID returns the URL for this domain/actor
func (service *Domain) ActorID() string {
	return dt.AddProtocol(service.hostname) + "/@application"
}

// PublicKeyID returns the URL for the public key for this domain/actor
func (service *Domain) PublicKeyID() string {
	return service.ActorID() + "#main-key"
}

// PublicKeyPEM returns the PEM-encoded public key for this domain/actor
func (service *Domain) PublicKeyPEM(session data.Session) (string, error) {

	// Try to retrieve the private key for this domain
	privateKey, err := service.PrivateKey(session)

	if err != nil {
		return "", derp.Wrap(err, "service.Domain.PublicKeyPEM", "Error getting public key")
	}

	// Encode the public key portion
	publicKeyPEM := sigs.EncodePublicPEM(privateKey)
	return publicKeyPEM, nil
}

// PrivateKey returns the private key for this domain/actor
func (service *Domain) PrivateKey(session data.Session) (*rsa.PrivateKey, error) {

	const location = "service.Domain.PrivateKey"

	// Get the Domain record
	domain := *service.Get()

	// Try to use the existing private key
	if domain.PrivateKey != "" {

		privateKey, err := sigs.DecodePrivatePEM(domain.PrivateKey)

		if err == nil {
			if rsaKey, ok := privateKey.(*rsa.PrivateKey); ok {
				return rsaKey, nil
			}
		}

		// Fall through means that we have a value for "domain.PrivateKey" but it's not
		// valid.  So, let's log the error and try to make a new one.
		derp.Report(derp.Wrap(err, location, "Unable to decode private key. Creating a new key"))
	}

	// Otherwise, create a new private key, save it, and return it to the caller.
	privateKey, err := rsa.GenerateKey(rand.Reader, encryptionKeyBits)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to generate RSA key")
	}

	// Save the new private key into the Domain record
	domain.PrivateKey = sigs.EncodePrivatePEM(privateKey)

	if err := service.Save(session, domain, "Generated Private Key"); err != nil {
		return nil, derp.Wrap(err, location, "Unable to save new EncryptionKey")
	}

	// Success??
	return privateKey, nil
}

// ActivityPubActor returns an ActivityPub Actor object
// ** WHICH INCLUDES ENCRYPTION KEYS ** for the provided User.
func (service *Domain) ActivityPubActor(session data.Session) (outbox.Actor, error) {

	const location = "service.Domain.ActivityPubActor"

	// Retrieve the Private Key from the Domain record
	privateKey, err := service.PrivateKey(session)

	if err != nil {
		return outbox.Actor{}, derp.Wrap(err, location, "Error extracting private key")
	}

	// Return the ActivityPub Actor
	actor := outbox.NewActor(
		service.ActorID(),
		privateKey,
		outbox.WithClient(service.activityService.AppClient()),
	)

	return actor, nil
}

func (service *Domain) WebFinger() digit.Resource {

	// Make a WebFinger resource for this Stream.
	result := digit.NewResource("acct:application@"+service.Hostname()).
		Alias(service.ActorID()).
		Link(digit.RelationTypeSelf, model.MimeTypeActivityPub, service.ActorID()).
		Link(digit.RelationTypeProfile, model.MimeTypeHTML, service.ActorID())

	return result
}
