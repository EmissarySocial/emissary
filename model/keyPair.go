package model

type KeyPair struct {
	PublicKey  string `json:"publicKey" bson:"publicKey"`
	PrivateKey string `json:"privateKey" bson:"privateKey"`
}

func NewKeyPair() KeyPair {
	return KeyPair{}
}
