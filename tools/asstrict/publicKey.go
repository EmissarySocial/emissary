package asstrict

import (
	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

type PublicKey struct {
	ID           string `json:"id"           bson:"id"`
	PublicKeyPEM string `json:"publicKeyPEM" bson:"publicKeyPEM"`
}

func NewPublicKey(value any) PublicKey {

	var object mapof.Any = convert.MapOfAny(value)

	return PublicKey{
		ID:           object.GetString(vocab.PropertyID),
		PublicKeyPEM: object.GetString(vocab.PropertyPublicKeyPEM),
	}
}

// Get returns a value of the given property
func (publicKey PublicKey) Get(name string) property.Value {

	switch name {

	case vocab.PropertyID:
		return property.String(publicKey.ID)

	case vocab.PropertyPublicKeyPEM:
		return property.String(publicKey.PublicKeyPEM)
	}

	return property.Nil{}
}

// Set returns the value with the given property set
func (publicKey PublicKey) Set(name string, value any) property.Value {

	switch name {

	case vocab.PropertyID:
		publicKey.ID = convert.String(value)

	case vocab.PropertyPublicKeyPEM:
		publicKey.PublicKeyPEM = convert.String(value)
	}

	return publicKey
}

// Head returns the first value in a slices, or the value itself if it is not a slice
func (publicKey PublicKey) Head() property.Value {
	return publicKey
}

// Tail returns all values in a slice except the first
func (publicKey PublicKey) Tail() property.Value {
	return property.Nil{}
}

// Len returns the number of elements in the value
func (publicKey PublicKey) Len() int {
	return 1
}

// IsNil returns TRUE if the value is empty
func (publicKey PublicKey) IsNil() bool {
	return false
}

// Map returns the map representation of this value
func (publicKey PublicKey) Map() map[string]any {

	return map[string]any{
		vocab.PropertyID:           publicKey.ID,
		vocab.PropertyPublicKeyPEM: publicKey.PublicKeyPEM,
	}
}

// Raw returns the raw, unwrapped value being stored
func (publicKey PublicKey) Raw() any {
	return publicKey
}

// Clone returns a deep copy of a value
func (publicKey PublicKey) Clone() property.Value {
	return publicKey
}
