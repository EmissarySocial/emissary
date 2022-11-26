package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EncryptionKey defines a service that tracks the (possibly external) accounts an internal User is encryptionKey.
type EncryptionKey struct {
	collection data.Collection
}

// NewEncryptionKey returns a fully initialized EncryptionKey service
func NewEncryptionKey(collection data.Collection) EncryptionKey {
	service := EncryptionKey{}
	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *EncryptionKey) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *EncryptionKey) Close() {
	// Nothin to do here.
}

/*******************************************
 * Common Data Methods
 *******************************************/

// List returns an iterator containing all of the EncryptionKeys who match the provided criteria
func (service *EncryptionKey) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an EncryptionKey from the database
func (service *EncryptionKey) Load(criteria exp.Expression, encryptionKey *model.EncryptionKey) error {

	if err := service.collection.Load(notDeleted(criteria), encryptionKey); err != nil {
		return derp.Wrap(err, "service.EncryptionKey.Load", "Error loading EncryptionKey", criteria)
	}

	return nil
}

// Save adds/updates an EncryptionKey in the database
func (service *EncryptionKey) Save(encryptionKey *model.EncryptionKey, note string) error {

	if err := service.collection.Save(encryptionKey, note); err != nil {
		return derp.Wrap(err, "service.EncryptionKey.Save", "Error saving EncryptionKey", encryptionKey, note)
	}

	return nil
}

// Delete removes an EncryptionKey from the database (virtual delete)
func (service *EncryptionKey) Delete(encryptionKey *model.EncryptionKey, note string) error {

	// Delete this EncryptionKey
	if err := service.collection.Delete(encryptionKey, note); err != nil {
		return derp.Wrap(err, "service.EncryptionKey.Delete", "Error deleting EncryptionKey", encryptionKey, note)
	}

	return nil
}

/*******************************************
 * Model Service Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *EncryptionKey) ObjectType() string {
	return "EncryptionKey"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *EncryptionKey) ObjectNew() data.Object {
	result := model.NewEncryptionKey()
	return &result
}

func (service *EncryptionKey) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.EncryptionKey); ok {
		return mention.EncryptionKeyID
	}

	return primitive.NilObjectID
}

func (service *EncryptionKey) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, criteria, options...)
}

func (service *EncryptionKey) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *EncryptionKey) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewEncryptionKey()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *EncryptionKey) ObjectSave(object data.Object, comment string) error {
	if encryptionKey, ok := object.(*model.EncryptionKey); ok {
		return service.Save(encryptionKey, comment)
	}
	return derp.NewInternalError("service.EncryptionKey.ObjectSave", "Invalid Object Type", object)
}

func (service *EncryptionKey) ObjectDelete(object data.Object, comment string) error {
	if encryptionKey, ok := object.(*model.EncryptionKey); ok {
		return service.Delete(encryptionKey, comment)
	}
	return derp.NewInternalError("service.EncryptionKey.ObjectDelete", "Invalid Object Type", object)
}

func (service *EncryptionKey) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.EncryptionKey", "Not Authorized")
}

func (service *EncryptionKey) Schema() schema.Schema {
	return schema.New(model.EncryptionKeySchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *EncryptionKey) GetPrivateKey(userID primitive.ObjectID) (*rsa.PrivateKey, error) {

	// Load the encryption key
	encryptionKey := model.NewEncryptionKey()
	criteria := exp.Equal("userID", userID)

	if err := service.Load(criteria, &encryptionKey); err != nil {
		return nil, derp.Wrap(err, "service.EncryptionKey.GetPrivateKey", "Error loading encryption key", userID)
	}

	// Decode the private key
	return service.getPrivateKey(&encryptionKey)
}

func (service *EncryptionKey) GetPublicKey(userID primitive.ObjectID) (*rsa.PublicKey, error) {

	// Load the encryption key
	encryptionKey := model.NewEncryptionKey()
	criteria := exp.Equal("userID", userID)

	if err := service.Load(criteria, &encryptionKey); err != nil {
		return nil, derp.Wrap(err, "service.EncryptionKey.GetPrivateKey", "Error loading encryption key", userID)
	}

	// Decode the private key
	return service.getPublicKey(&encryptionKey)
}

/*******************************************
 * Custom Actions
 *******************************************/

func (service *EncryptionKey) Create(userID primitive.ObjectID) (model.EncryptionKey, error) {

	// Create new model object
	encryptionKey := model.NewEncryptionKey()
	encryptionKey.UserID = userID
	encryptionKey.Encoding = model.EncryptionKeyEncodingPlaintext // TODO: MEDIUM: add key encryption encoding

	// Create an actual encryption key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return model.EncryptionKey{}, derp.Wrap(err, "model.CreateEncryptionKey", "Error generating RSA key", userID)
	}

	encryptionKey.PrivatePEM = service.encodePrivatePEM(privateKey)
	encryptionKey.PublicPEM = service.encodePublicPEM(privateKey)

	return encryptionKey, nil
}

/******************************
 * Data Accessors
 ******************************/

func (service *EncryptionKey) Sign(message []byte, encryptionKey *model.EncryptionKey) ([]byte, error) {

	privateKey, err := service.getPrivateKey(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, "model.EncryptionKey.Sign", "Error getting private key", encryptionKey.EncryptionKeyID)
	}

	return rsa.SignPKCS1v15(rand.Reader, privateKey, 0, message)
}

func (service *EncryptionKey) Verify(message []byte, signature []byte, encryptionKey *model.EncryptionKey) error {

	publicKey, err := service.getPublicKey(encryptionKey)

	if err != nil {
		return derp.Wrap(err, "model.EncryptionKey.Validate", "Error getting public key", encryptionKey.EncryptionKeyID)
	}

	return rsa.VerifyPKCS1v15(publicKey, 0, message, signature)
}

func (service *EncryptionKey) getPublicKey(encryptionKey *model.EncryptionKey) (*rsa.PublicKey, error) {

	privateKey, err := service.getPrivateKey(encryptionKey)

	if err != nil {
		return nil, derp.Wrap(err, "model.EncryptionKey.PublicKey", "Error getting private key", encryptionKey.EncryptionKeyID)
	}

	return &privateKey.PublicKey, nil
}

func (service *EncryptionKey) getPrivateKey(encryptionKey *model.EncryptionKey) (*rsa.PrivateKey, error) {

	// Decode PEM block
	block, _ := pem.Decode([]byte(encryptionKey.PrivatePEM))

	// Parse the key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)

	if err != nil {
		return nil, derp.Wrap(err, "model.EncryptionKey.PrivateKey", "Error parsing private key", encryptionKey.EncryptionKeyID)
	}

	return privateKey, nil
}

/******************************
 * Helper Methods
 ******************************/

func (service *EncryptionKey) encodePrivatePEM(privateKey *rsa.PrivateKey) string {

	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return string(privatePEM)
}

func (service *EncryptionKey) encodePublicPEM(privateKey *rsa.PrivateKey) string {

	// Get ASN.1 DER format
	publicDER := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

	// pem.Block
	publicBlock := pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   publicDER,
	}

	// Private key in PEM format
	publicPEM := pem.EncodeToMemory(&publicBlock)

	return string(publicPEM)
}
