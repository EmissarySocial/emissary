package model

// ResourceKey represents one decryption key
type ResourceKey struct {
	ResourceKeyID string `json:"resourceId" bson:"_id"`
	Label         string `json:"label"      bson:"label"`
	EncryptedKey
}
