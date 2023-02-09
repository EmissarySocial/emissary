package activitypub

import (
	"crypto"
)

// Actor defines a local actor that can send and receive ActivityStream messages.
// Apps should populate this struct and pass it into the middleware functions.
type Actor struct {
	ActorID     string
	PublicKeyID string
	PublicKey   crypto.PublicKey
	PrivateKey  crypto.PrivateKey
}
