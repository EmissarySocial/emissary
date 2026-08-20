package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/sigs"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Require 2048-bit encryption keys
const encryptionKeyBits = 2048

// EncryptionKey defines a service that tracks the (possibly external) accounts an internal User is encryptionKey.
type EncryptionKey struct {
	host string
}

// NewEncryptionKey returns a fully initialized EncryptionKey service
func NewEncryptionKey() EncryptionKey {
	return EncryptionKey{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *EncryptionKey) Refresh(factory *Factory) {
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *EncryptionKey) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the EncryptionKey collection for the provided database session
func (service *EncryptionKey) collection(session data.Session) data.Collection {
	return session.Collection("EncryptionKey")
}

// Count returns the number of records that match the provided criteria
func (service *EncryptionKey) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// List returns an iterator containing all of the EncryptionKeys who match the provided criteria
func (service *EncryptionKey) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns an iterator containing all of the EncryptionKeys that match the provided criteria
func (service *EncryptionKey) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.EncryptionKey], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.EncryptionKey.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewEncryptionKey), nil
}

// Load retrieves an EncryptionKey from the database
func (service *EncryptionKey) Load(session data.Session, criteria exp.Expression, encryptionKey *model.EncryptionKey) error {

	if err := service.collection(session).Load(notDeleted(criteria), encryptionKey); err != nil {
		return derp.Wrap(err, "service.EncryptionKey.Load", "Loading EncryptionKey", criteria)
	}

	return nil
}

// Save adds/updates an EncryptionKey in the database
func (service *EncryptionKey) Save(session data.Session, encryptionKey *model.EncryptionKey, note string) error {

	if err := service.collection(session).Save(encryptionKey, note); err != nil {
		return derp.Wrap(err, "service.EncryptionKey.Save", "Saving EncryptionKey", encryptionKey, note)
	}

	return nil
}

// Delete removes an EncryptionKey from the database (virtual delete)
func (service *EncryptionKey) Delete(session data.Session, encryptionKey *model.EncryptionKey, note string) error {

	// Delete this EncryptionKey
	if err := service.collection(session).Delete(encryptionKey, note); err != nil {
		return derp.Wrap(err, "service.EncryptionKey.Delete", "Deleting EncryptionKey", encryptionKey, note)
	}

	return nil
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeByParentID returns an iterator over every EncryptionKey belonging to the provided parent
func (service *EncryptionKey) RangeByParentID(session data.Session, parentID primitive.ObjectID) (iter.Seq[model.EncryptionKey], error) {
	return service.Range(session, exp.Equal("parentId", parentID))
}

// LoadByID tries to load the EncryptionKey from the database.  If no key
// exists for the designated user, then a new one is generated.
func (service *EncryptionKey) LoadByParentID(session data.Session, parentType string, parentID primitive.ObjectID, encryptionKey *model.EncryptionKey) error {

	const location = "service.EncryptionKey.LoadByParentID"

	// Try to load the encryption key from the database
	if err := service.Load(session, exp.Equal("parentType", parentType).AndEqual("parentId", parentID), encryptionKey); err != nil {

		// Anything other than NotFound is a real failure; a NotFound falls through to make a key
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Loading EncryptionKey", parentID)
		}
	} else {
		return nil
	}

	// Fall through means "Not Found", so create a new key
	newKey, err := service.Create(session, parentType, parentID)

	if err != nil {
		return derp.Wrap(err, location, "Creating new EncryptionKey", parentID)
	}

	// Return the key if successful
	*encryptionKey = newKey
	return nil
}

/******************************************
 * Custom Actions
 ******************************************/

// Create generates a new RSA key pair for the provided parent, and saves it
func (service *EncryptionKey) Create(session data.Session, parentType string, parentID primitive.ObjectID) (model.EncryptionKey, error) {

	// Create new model object
	encryptionKey := model.NewEncryptionKey()
	encryptionKey.ParentType = parentType
	encryptionKey.ParentID = parentID
	encryptionKey.Encoding = model.EncryptionKeyEncodingPlaintext // TODO: MEDIUM: add key encryption encoding

	// Create an actual encryption key
	privateKey, err := rsa.GenerateKey(rand.Reader, encryptionKeyBits)

	if err != nil {
		return model.EncryptionKey{}, derp.Wrap(err, "model.CreateEncryptionKey", "Generating RSA key", parentType, parentID)
	}

	encryptionKey.PrivatePEM = sigs.EncodePrivatePEM(privateKey)
	encryptionKey.PublicPEM = sigs.EncodePublicPEM(privateKey)

	if err := service.Save(session, &encryptionKey, "Created"); err != nil {
		return model.EncryptionKey{}, derp.Wrap(err, "model.CreateEncryptionKey", "Saving new EncryptionKey", parentType, parentID)
	}

	return encryptionKey, nil
}

// DeleteByParentID marks every EncryptionKey belonging to the provided parent as deleted
func (service *EncryptionKey) DeleteByParentID(session data.Session, parentID primitive.ObjectID, note string) error {

	const location = "service.EncryptionKey.DeleteByParentID"

	rangeFunc, err := service.RangeByParentID(session, parentID)

	if err != nil {
		return derp.Wrap(err, location, "Loading keys", parentID)
	}

	for encryptionKey := range rangeFunc {
		if err := service.Delete(session, &encryptionKey, note); err != nil {
			return derp.Wrap(err, location, "Deleting key", encryptionKey)
		}
	}

	return nil
}

/******************************************
 * Data Accessors
 ******************************************/

// GetPublicKey returns the RSA public key half of the provided EncryptionKey
func (service *EncryptionKey) GetPublicKey(encryptionKey *model.EncryptionKey) (*rsa.PublicKey, error) {

	const location = "service.EncryptionKey.PublicKey"

	privateKey, err := service.GetPrivateKey(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, location, "Getting private key", encryptionKey.EncryptionKeyID)
	}

	if privateKey == nil {
		return nil, derp.Internal(location, "Private key cannot be nil")
	}

	return &privateKey.PublicKey, nil
}

// GetPrivateKey returns the RSA private key half of the provided EncryptionKey
func (service *EncryptionKey) GetPrivateKey(encryptionKey *model.EncryptionKey) (*rsa.PrivateKey, error) {

	const location = "service.EncryptionKey.GetPrivateKey"

	// Decode PEM block
	block, _ := pem.Decode([]byte(encryptionKey.PrivatePEM))

	if block == nil {
		return nil, derp.Internal(location, "Unable to decode private key PEM", encryptionKey.EncryptionKeyID)
	}

	// Parse the key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)

	if err != nil {
		return nil, derp.Wrap(err, location, "Parsing private key", encryptionKey.EncryptionKeyID)
	}

	if privateKey == nil {
		return nil, derp.Wrap(err, location, "Private key cannot be nil")
	}

	return privateKey, nil
}

// Sign signs a message with the private half of the provided EncryptionKey
func (service *EncryptionKey) Sign(message []byte, encryptionKey *model.EncryptionKey) ([]byte, error) {

	privateKey, err := service.GetPrivateKey(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, "model.EncryptionKey.Sign", "Getting private key", encryptionKey.EncryptionKeyID)
	}

	return rsa.SignPKCS1v15(rand.Reader, privateKey, 0, message)
}

// Verify confirms that a signature was made by the private half of the provided EncryptionKey
func (service *EncryptionKey) Verify(message []byte, signature []byte, encryptionKey *model.EncryptionKey) error {

	publicKey, err := service.GetPublicKey(encryptionKey)

	if err != nil {
		return derp.Wrap(err, "model.EncryptionKey.Validate", "Getting public key", encryptionKey.EncryptionKeyID)
	}

	return rsa.VerifyPKCS1v15(publicKey, 0, message, signature)
}

/******************************************
 * Other Key Metadata
 ******************************************/

// OwnerID returns the publicly accessible URL of the Actor who owns this EncryptionKey
func (service *EncryptionKey) OwnerID(encryptionKey *model.EncryptionKey) string {

	if encryptionKey.ParentType == model.EncryptionKeyTypeUser {
		return service.host + "/@" + encryptionKey.ParentID.Hex()
	}

	return service.host + "/" + encryptionKey.ParentID.Hex()
}

// KeyID returns the publicly accessible URL of this EncryptionKey
func (service *EncryptionKey) KeyID(encryptionKey *model.EncryptionKey) string {
	return service.OwnerID(encryptionKey) + "#main-key"
}
