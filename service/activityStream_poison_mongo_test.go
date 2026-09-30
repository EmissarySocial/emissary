package service

import (
	"testing"

	"github.com/benpate/hannibal/clients"
	"github.com/stretchr/testify/require"
)

// These tests run hostile documents through the whole production stack, including ascache over a
// real replica set, because a poisoned entry only exists once the cache has written it.  They skip
// when no database is reachable, so `go test ./...` still passes on a machine without one.

// TestActivityStream_FullStack_ForeignDocumentCannotPlantAKey confirms that an actor served by one
// host, claiming another host's actor id, never becomes the key that signature verification reads.
func TestActivityStream_FullStack_ForeignDocumentCannotPlantAKey(t *testing.T) {

	// Cache poisoning: the planted key would verify activities that the attacker signs as the victim.
	database := newCycleTestDatabase(t)
	server := newCycleServer(t)

	// The victim lives at 127.0.0.1, which the cache treats as a different host from localhost
	victim := server.url("/actors/victim")
	server.serve("/actors/victim", keyedActor(victim, "PEM-GENUINE"))

	// The attacker, at localhost, serves a copy of the victim's actor carrying its own key
	server.serve("/actors/impostor", keyedActor(victim, "PEM-ATTACKER"))
	loadWithin(t, fullStack(server, clients.NewCarpool(), database), server.localhostURL("/actors/impostor"))

	// Loading the victim by its id, and its key by the keyId, must return the victim's own key
	actor := loadWithin(t, fullStack(server, clients.NewCarpool(), database), victim)
	require.Equal(t, "PEM-GENUINE", actor.PublicKey().PublicKeyPEM())

	key := loadWithin(t, fullStack(server, clients.NewCarpool(), database), victim+"#main-key")
	require.Equal(t, "PEM-GENUINE", key.PublicKeyPEM())
}

// keyedActor returns a minimal Person at the given id, publishing the given key at id#main-key
func keyedActor(id string, publicKeyPEM string) map[string]any {

	result := cycleActor(id)
	result["publicKey"] = map[string]any{
		"id":           id + "#main-key",
		"owner":        id,
		"publicKeyPem": publicKeyPEM,
	}

	return result
}
