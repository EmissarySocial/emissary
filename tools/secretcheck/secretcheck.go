package secretcheck

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
)

// Find returns the forms of `err` that contain `secret`: any of "message", "json", and "bson".
// It returns an error when a form cannot be encoded, because an unchecked form proves nothing.
func Find(err error, secret string) ([]string, error) {

	const location = "secretcheck.Find"

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
