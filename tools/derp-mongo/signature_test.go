package derpmongo

import (
	"errors"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/assert"
)

// signatureTextLength is the full width of a signature, including its version prefix
const signatureTextLength = len(signatureVersion) + 1 + signatureLength

func TestNormalizeMessage(t *testing.T) {

	tests := map[string]string{
		"lookup bsky.brid.gy: no such host":       "lookup <host>: no such host",
		"Loading https://x.com/users/1 failed":    "Loading <url> failed",
		"object 6aa17bd9a102749e24915ab4 is gone": "object <id> is gone",
		"timed out after 30000 ms":                "timed out after <n> ms",
		"  collapse   the   spaces  ":             "collapse the spaces",
		"":                                        "",
		"context deadline exceeded":               "context deadline exceeded",
	}

	for input, expected := range tests {
		assert.Equal(t, expected, normalizeMessage(input), "normalizing %q", input)
	}
}

func TestNormalizeMessage_KeepsGoIdentifiers(t *testing.T) {

	// A Go type looks like a hostname but must survive, or two unrelated template errors
	// would collapse into one signature and only the first would ever be investigated.
	input := `can't evaluate field Object in type build.Follower`
	assert.Equal(t, input, normalizeMessage(input))

	assert.Equal(t, "service.Widget.Save failed", normalizeMessage("service.Widget.Save failed"))
}

func TestNormalizeMessage_ShortNumbersSurvive(t *testing.T) {

	// Line and column numbers distinguish two errors in the same template
	assert.Equal(t, "template: x:1:17: bad", normalizeMessage("template: x:1:17: bad"))
	assert.Equal(t, "403 Forbidden", normalizeMessage("403 Forbidden"))
}

func TestNormalizeMessage_IsSinglePass(t *testing.T) {

	// Normalizing is NOT idempotent, and Signature relies on calling it exactly once.  A
	// placeholder rewrites the word boundaries around it, which exposes matches the first pass
	// could not reach: below, "<n>" puts a boundary in front of text that then reads as a host.
	once := normalizeMessage("A000000000000a.aaaaa")
	assert.Equal(t, "A<n>a.aaaaa", once)
	assert.Equal(t, "A<n><host>", normalizeMessage(once), "a second pass matches more, which is why there is never one")
}

func TestNormalizeMessage_InvalidUTF8(t *testing.T) {
	assert.NotPanics(t, func() {
		normalizeMessage("\x00\xff\xfe broken \xc3")
	})
}

func TestSignature_IsStable(t *testing.T) {

	first := signature(500, "service.A", "service.B", "lookup one.example.com: no such host")
	second := signature(500, "service.A", "service.B", "lookup two.example.org: no such host")

	assert.Equal(t, first, second, "the same failure against two hosts is one error")
	assert.Len(t, first, signatureTextLength)
}

func TestSignature_SeparatesDifferentErrors(t *testing.T) {

	base := signature(500, "service.A", "service.B", "broken")

	assert.NotEqual(t, base, signature(404, "service.A", "service.B", "broken"))
	assert.NotEqual(t, base, signature(500, "service.Z", "service.B", "broken"))
	assert.NotEqual(t, base, signature(500, "service.A", "service.Z", "broken"))
	assert.NotEqual(t, base, signature(500, "service.A", "service.B", "different"))
}

func TestSignature_HandlesEmptyInput(t *testing.T) {
	assert.Len(t, signature(0, "", "", ""), signatureTextLength)
}

func TestSignature_HandlesExtremeInput(t *testing.T) {

	huge := strings.Repeat("a.b/", 20000)

	assert.Len(t, signature(-1, huge, huge, huge), signatureTextLength)
	assert.Len(t, signature(1<<62, "\x00", "\xff\xfe", "\x00"), signatureTextLength)
}

func TestSignature_FieldsCannotBleedTogether(t *testing.T) {

	// The separator must keep "ab"+"c" apart from "a"+"bc"
	assert.NotEqual(t,
		signature(500, "ab", "c", "x"),
		signature(500, "a", "bc", "x"),
	)
}

func TestSignature_CarriesVersionPrefix(t *testing.T) {

	value := signature(500, "service.A", "service.B", "broken")

	assert.True(t, strings.HasPrefix(value, signatureVersion+":"), "a signature names the algorithm that made it")
	assert.Len(t, strings.TrimPrefix(value, signatureVersion+":"), signatureLength)
	assert.Regexp(t, `^v1:[0-9a-f]{12}$`, value)
}

func TestSignatureOf_ReadsTheWholeChain(t *testing.T) {

	inner := derp.NotFound("service.Inner.Load", "record not found")
	outer := derp.Wrap(inner, "service.Outer.Do", "could not do the thing")

	// The four seed values come from derp, and this pins which accessor feeds which field
	assert.Equal(t, 404, derp.ErrorCode(outer), "Wrap bubbles the inner status code up")
	assert.Equal(t, "service.Outer.Do", derp.Location(outer), "the origin is where the failure surfaced")
	assert.Equal(t, "service.Inner.Load", derp.RootLocation(outer))
	assert.Equal(t, "record not found", derp.RootMessage(outer))

	assert.Equal(t, signature(404, "service.Outer.Do", "service.Inner.Load", "record not found"), SignatureOf(outer))
}

func TestSignatureOf_OriginSeparatesCallSites(t *testing.T) {

	inner := derp.NotFound("service.Inner.Load", "record not found")

	first := SignatureOf(derp.Wrap(inner, "service.First.Do", "failed"))
	second := SignatureOf(derp.Wrap(inner, "service.Second.Do", "failed"))

	assert.NotEqual(t, first, second, "one root cause reached from two call sites is two items of work")
}

func TestSignatureOf_SeparatesPlainErrors(t *testing.T) {

	// Reading these back out of BSON cannot tell them apart, because a plain error marshals
	// to an empty document.  Signing the live error is what keeps them separate.
	first := SignatureOf(errors.New("boom"))
	second := SignatureOf(errors.New("totally different message"))

	assert.NotEqual(t, first, second)
	assert.Equal(t, signature(500, "", "", "boom"), first, "a plain error has no location and a generic status")
}

func TestSignatureOf_HandlesNil(t *testing.T) {

	// derp.Report never calls a reporter with a nil error, but nothing here may panic if it does
	assert.NotPanics(t, func() {
		assert.Equal(t, signature(0, "", "", ""), SignatureOf(nil))
	})
}

func TestSignatureOf_IgnoresVariableDetail(t *testing.T) {

	first := derp.Internal("service.A", "lookup one.example.com: no such host")
	second := derp.Internal("service.A", "lookup two.example.net: no such host")

	assert.Equal(t, SignatureOf(first), SignatureOf(second), "the same defect against two hosts is one item")
}

func FuzzNormalizeMessage(f *testing.F) {

	f.Add("lookup bsky.brid.gy: no such host")
	f.Add("https://x.com/a?b=c")
	f.Add("")
	f.Add("....")
	f.Add("\x00\xff")
	f.Add("0000://0")
	f.Add("0.0000")
	f.Add(strings.Repeat("a.b", 2000))

	f.Fuzz(func(t *testing.T, input string) {

		result := normalizeMessage(input)

		// The same message must always normalize the same way, or one error would sign
		// as two and be investigated twice.
		assert.Equal(t, result, normalizeMessage(input), "normalizing %q must be deterministic", input)

		// Whitespace is always collapsed, so a reformatted message is still one error
		assert.Equal(t, strings.TrimSpace(result), result, "normalizing %q left outer whitespace", input)
		assert.NotRegexp(t, `\s\s`, result, "normalizing %q left a run of whitespace", input)
	})
}

func FuzzSignature(f *testing.F) {

	f.Add(500, "a", "b", "c")
	f.Add(0, "", "", "")
	f.Add(-1, "|", "|", "|")
	f.Add(1<<62, "\x00", "\xff", "https://x.com/0000")

	f.Fuzz(func(t *testing.T, code int, origin string, root string, message string) {

		first := signature(code, origin, root, message)

		assert.Len(t, first, signatureTextLength)
		assert.Equal(t, first, signature(code, origin, root, message), "signatures must be deterministic")
		assert.True(t, strings.HasPrefix(first, signatureVersion+":"))
	})
}
