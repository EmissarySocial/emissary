package secretcheck

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// hiddenFromJSON mimics the models that hide a secret from JSON but not from BSON
type hiddenFromJSON struct {
	Name   string `json:"name"   bson:"name"`
	Secret string `json:"-"      bson:"secret"`
}

// visibleEverywhere mimics the models that hide nothing, such as config.Domain
type visibleEverywhere struct {
	Secret string `json:"secret" bson:"secret"`
}

// fatalRecorder is a testing.TB that records Fatalf instead of stopping the test
type fatalRecorder struct {
	testing.TB
	message string
}

// Helper satisfies testing.TB
func (recorder *fatalRecorder) Helper() {}

// Fatalf records the failure message
func (recorder *fatalRecorder) Fatalf(format string, args ...any) {
	recorder.message = fmt.Sprintf(format, args...)
}

// TestFind_Nil verifies that a nil error carries nothing
func TestFind_Nil(t *testing.T) {
	requireForms(t, nil, "s3cr3t")
}

// TestFind_Clean verifies that an error without the secret reports no forms
func TestFind_Clean(t *testing.T) {
	requireForms(t, derp.Internal("test", "Loading domain", "example.com"), "s3cr3t")
}

// TestFind_Message verifies that a secret in the message is found in every form
func TestFind_Message(t *testing.T) {
	requireForms(t, derp.Internal("test", "Bad key s3cr3t"), "s3cr3t", "message", "json", "bson")
}

// TestFind_VisibleDetail verifies that a detail with no json:"-" leaks through both encodings
func TestFind_VisibleDetail(t *testing.T) {
	err := derp.Internal("test", "Loading domain", visibleEverywhere{Secret: "s3cr3t"})
	requireForms(t, err, "s3cr3t", "json", "bson")
}

// TestFind_HiddenFromJSON verifies the case this package exists for: a json:"-" field is
// invisible to a JSON check and is still stored
func TestFind_HiddenFromJSON(t *testing.T) {

	// OAuthClient.ClientSecret and Connection.Token have this shape (BUG-173)
	err := derp.Internal("test", "Saving client", hiddenFromJSON{Name: "app", Secret: "s3cr3t"})
	requireForms(t, err, "s3cr3t", "bson")
}

// TestFind_PointerDetail verifies that a pointer detail is encoded the same as a value
func TestFind_PointerDetail(t *testing.T) {
	err := derp.Internal("test", "Saving client", &hiddenFromJSON{Secret: "s3cr3t"})
	requireForms(t, err, "s3cr3t", "bson")
}

// TestFind_Wrapped verifies that a detail is found at any depth of derp wrapping
func TestFind_Wrapped(t *testing.T) {

	inner := derp.Internal("inner", "Connecting", hiddenFromJSON{Secret: "s3cr3t"})
	outer := derp.Wrap(derp.Wrap(inner, "middle", "Refreshing"), "outer", "Starting")

	requireForms(t, outer, "s3cr3t", "bson")
}

// TestFind_ForeignWrapper pins what a non-derp wrapper hides from every reporter
func TestFind_ForeignWrapper(t *testing.T) {

	// fmt's wrapper has only unexported fields, so neither encoding descends into it.  The
	// reporters store the same encodings, so the secret below is not stored either.
	inner := derp.Internal("inner", "Connecting", hiddenFromJSON{Secret: "s3cr3t"})
	requireForms(t, fmt.Errorf("wrapped: %w", inner), "s3cr3t")
}

// TestFind_HTMLCharacters verifies that a secret with HTML-special characters is matched as written
func TestFind_HTMLCharacters(t *testing.T) {
	err := derp.Internal("test", "Connecting", visibleEverywhere{Secret: "a&b<c>d"})
	requireForms(t, err, "a&b<c>d", "json", "bson")
}

// TestFind_PlainError verifies that a non-derp error is checked by its message
func TestFind_PlainError(t *testing.T) {
	requireForms(t, errors.New("dial tcp: s3cr3t"), "s3cr3t", "message")
}

// requireForms asserts which forms Find reports
func requireForms(t *testing.T, err error, secret string, expected ...string) {

	t.Helper()

	forms, findErr := Find(err, secret)
	require.NoError(t, findErr)

	if len(expected) == 0 {
		require.Empty(t, forms)
		return
	}

	require.Equal(t, expected, forms)
}

// TestFind_Unencodable verifies that a detail no encoder accepts is reported, not ignored
func TestFind_Unencodable(t *testing.T) {

	forms, err := Find(derp.Internal("test", "Bad detail", func() {}), "s3cr3t")

	require.Error(t, err)
	require.Nil(t, forms)
}

// TestFind_UnencodableAsBSON verifies that a detail only JSON accepts is still reported
func TestFind_UnencodableAsBSON(t *testing.T) {

	// BSON has no unsigned 64-bit type, so this value encodes as JSON and fails as BSON
	forms, err := Find(derp.Internal("test", "Big number", uint64(math.MaxUint64)), "s3cr3t")

	require.Error(t, err)
	require.Contains(t, err.Error(), "BSON")
	require.Nil(t, forms)
}

// TestRequireAbsent_Passes verifies that a clean error does not fail the test
func TestRequireAbsent_Passes(t *testing.T) {

	recorder := &fatalRecorder{}
	RequireAbsent(recorder, derp.Internal("test", "Loading domain", "example.com"), "s3cr3t")

	require.Empty(t, recorder.message)
}

// TestRequireAbsent_Fails verifies that a leak fails the test without printing the secret
func TestRequireAbsent_Fails(t *testing.T) {

	recorder := &fatalRecorder{}
	RequireAbsent(recorder, derp.Internal("server.putDomain", "Saving", hiddenFromJSON{Secret: "s3cr3t"}), "s3cr3t")

	require.Contains(t, recorder.message, "bson")
	require.Contains(t, recorder.message, "server.putDomain")
	require.NotContains(t, recorder.message, "s3cr3t")
}

// TestRequireAbsent_EmptySecret verifies that an empty secret is refused
func TestRequireAbsent_EmptySecret(t *testing.T) {

	recorder := &fatalRecorder{}
	RequireAbsent(recorder, derp.Internal("test", "Loading domain"), "")

	require.Contains(t, recorder.message, "must not be empty")
}

// TestRequireAbsent_Unencodable verifies that an error that cannot be checked fails the test
func TestRequireAbsent_Unencodable(t *testing.T) {

	recorder := &fatalRecorder{}
	RequireAbsent(recorder, derp.Internal("test", "Bad detail", make(chan int)), "s3cr3t")

	require.Contains(t, recorder.message, "unable to check")
}

// plainError is a non-derp error that encodes to nothing, like the mongo driver's errors
type plainError struct{ message string }

// Error returns the message
func (err plainError) Error() string { return err.message }

// unencodableError is a non-derp error that neither JSON nor BSON can encode
type unencodableError struct{ Channel chan int }

// Error returns a fixed message
func (err unencodableError) Error() string { return "connection refused" }

// TestRequireAbsentFromEachLayer_Passes verifies that a clean chain passes, even when it cannot
// be encoded whole
func TestRequireAbsentFromEachLayer_Passes(t *testing.T) {

	// Like the mongo driver's errors, this layer holds a value no encoder accepts, so the whole
	// chain would fail RequireAbsent outright
	outer := derp.Wrap(unencodableError{Channel: make(chan int)}, "outer", "Starting", "example.com")

	recorder := &fatalRecorder{}
	RequireAbsentFromEachLayer(recorder, outer, "s3cr3t")
	require.Empty(t, recorder.message)
}

// TestRequireAbsentFromEachLayer_FindsADerpLayer verifies that a secret in any derp layer fails
func TestRequireAbsentFromEachLayer_FindsADerpLayer(t *testing.T) {

	inner := derp.Internal("inner", "Connecting", hiddenFromJSON{Secret: "s3cr3t"})
	outer := derp.Wrap(inner, "outer", "Starting")

	recorder := &fatalRecorder{}
	RequireAbsentFromEachLayer(recorder, outer, "s3cr3t")
	require.Contains(t, recorder.message, "inner")
}

// TestRequireAbsentFromEachLayer_FindsAPlainMessage verifies that a secret in a non-derp message fails
func TestRequireAbsentFromEachLayer_FindsAPlainMessage(t *testing.T) {

	outer := derp.Wrap(plainError{message: "dial s3cr3t"}, "outer", "Starting")

	recorder := &fatalRecorder{}
	RequireAbsentFromEachLayer(recorder, outer, "s3cr3t")
	require.Contains(t, recorder.message, "plainError")
	require.NotContains(t, recorder.message, "s3cr3t")
}

// ExampleFind shows a secret hidden from JSON that is still stored as BSON
func ExampleFind() {

	// A field tagged json:"-" is left out of the console, and kept by the error log
	err := derp.Internal("example", "Saving client", hiddenFromJSON{Name: "app", Secret: "s3cr3t"})

	forms, _ := Find(err, "s3cr3t")
	fmt.Println(forms)

	// Output: [bson]
}
