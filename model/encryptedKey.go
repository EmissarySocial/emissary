package model

import "github.com/benpate/data/journal"

// EncryptedKey represents an encryption key that has itself, been encrypted.
// It is not available to this server, but may be re-encrypted by its owner at some point in the future.
type EncryptedKey struct {
	EncryptedKeyID string
	Algorithm      string
	Value          string
	EncryptedDate  int64

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the unique identifier of this object, and fulfills part of the data.Object interface
func (encryptedKey *EncryptedKey) ID() string {
	return encryptedKey.EncryptedKeyID
}
