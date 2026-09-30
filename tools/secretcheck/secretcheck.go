package secretcheck

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// RequireAbsentFromEachLayer is RequireAbsent for an error chain that cannot be encoded whole.
// It checks every derp layer in every form, and every other layer by its message.
func RequireAbsentFromEachLayer(t testing.TB, err error, secret string) {

	t.Helper()

	// A mongo driver error anywhere in a chain is the usual reason it cannot be encoded

	for ; err != nil; err = errors.Unwrap(err) {

		// A derp layer is encoded on its own, so an unencodable layer beneath it cannot hide it
		if layer, isDerp := err.(derp.Error); isDerp {
			layer.WrappedValue = nil
			RequireAbsent(t, layer, secret)
			continue
		}

		// Any other layer is reported by its message, so that is all it can carry
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("secretcheck: a %T in the chain carries a secret in its message", err)
			return
		}
	}
}

// RequireAbsent fails the test immediately if `secret` appears in any form of `err` that an
// error reporter would print or store.
func RequireAbsent(t testing.TB, err error, secret string) {

	t.Helper()

	// RULE: An empty secret is contained in every string, so it can only report a false leak
	if secret == "" {
		t.Fatalf("secretcheck: the secret to search for must not be empty")
		return
	}

	forms, findErr := Find(err, secret)

	if findErr != nil {
		t.Fatalf("secretcheck: unable to check the error: %v", findErr)
		return
	}

	// The error itself is not printed, because that would copy the secret into the test log
	if len(forms) > 0 {
		t.Fatalf("secretcheck: error from %q carries a secret in its %s form", derp.Location(err), strings.Join(forms, ", "))
	}
}

// Find returns the forms of `err` that contain `secret`: any of "message", "json", and "bson",
// or an error when a form cannot be encoded
func Find(err error, secret string) ([]string, error) {

	const location = "secretcheck.Find"

	// A form that cannot be encoded is an error, never a pass, because an unchecked form
	// proves nothing

	// Nothing to leak.  Empty secrets are rejected by RequireAbsent, not here.
	if err == nil {
		return nil, nil
	}

	result := make([]string, 0, 3)

	// Check the message, which every reporter prints
	if strings.Contains(err.Error(), secret) {
		result = append(result, "message")
	}

	// Check the JSON that derp-console prints.  HTML escaping is off so that a secret
	// containing "&" or "<" is matched as written, not as & or <.
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)

	if encodeErr := encoder.Encode(err); encodeErr != nil {
		return nil, derp.Wrap(encodeErr, location, "Encoding error as JSON")
	}

	if bytes.Contains(encoded.Bytes(), []byte(secret)) {
		result = append(result, "json")
	}

	// RULE: This MUST mirror derp-mongo, which stores the error in a field of a document.
	// BSON follows bson tags, so this is the form that catches a `json:"-"` field.
	stored, bsonErr := bson.Marshal(bson.M{"error": err})

	if bsonErr != nil {
		return nil, derp.Wrap(bsonErr, location, "Encoding error as BSON")
	}

	if bytes.Contains(stored, []byte(secret)) {
		result = append(result, "bson")
	}

	return result, nil
}
