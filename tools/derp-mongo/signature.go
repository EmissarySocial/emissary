package derpmongo

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"

	"github.com/benpate/derp"
)

// signatureVersion identifies the algorithm that produced a signature
const signatureVersion = "v1"

// signatureLength is how much of the hash identifies an error
const signatureLength = 12

// Patterns that replace the parts of an error message that change from one occurrence to the next
var (
	variableURL      = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s"'<>]+`)
	variableObjectID = regexp.MustCompile(`\b[0-9a-f]{24}\b`)
	variableHostname = regexp.MustCompile(`\b[a-z0-9][a-z0-9-]*(?:\.[a-z0-9][a-z0-9-]*)*\.[a-z]{2,24}\b`)
	variableNumber   = regexp.MustCompile(`\d{4,}`)
	repeatedSpace    = regexp.MustCompile(`\s+`)
)

// Placeholders that stand in for the parts of a message that change between occurrences
const (
	placeholderURL      = "<url>"
	placeholderObjectID = "<id>"
	placeholderHostname = "<host>"
	placeholderNumber   = "<n>"
)

// SignatureOf returns the stable identity of an error, so that every occurrence of one
// defect is recognized as a single item of work.
func SignatureOf(err error) string {
	return Signature(derp.ErrorCode(err), derp.Location(err), derp.RootLocation(err), derp.RootMessage(err))
}

// Signature builds the stable identity of an error from the four values that describe it
func Signature(statusCode int, origin string, rootLocation string, rootMessage string) string {

	// Join the four values that identify this error.  The separator matters, because it is
	// what keeps two adjacent fields from bleeding into one another.
	seed := strings.Join([]string{
		strconv.Itoa(statusCode),
		origin,
		rootLocation,
		NormalizeMessage(rootMessage),
	}, "|")

	// Hash the seed down to something short enough to type on a command line
	sum := sha256.Sum256([]byte(seed))

	// Stamp the version onto the result, so that an algorithm change is visible on disk
	return signatureVersion + ":" + hex.EncodeToString(sum[:])[:signatureLength]
}

// NormalizeMessage removes the parts of an error message that vary between occurrences, so
// that the same failure against two different hosts still produces one signature.
func NormalizeMessage(message string) string {

	// RULE: Single-pass only.  Each replacement rewrites the word boundaries around it, so a
	// second pass matches text the first could not reach.  Order matters here too: a URL
	// contains both hostnames and digits, so it is replaced whole, and first.
	message = variableURL.ReplaceAllString(message, placeholderURL)
	message = variableObjectID.ReplaceAllString(message, placeholderObjectID)
	message = variableHostname.ReplaceAllString(message, placeholderHostname)
	message = variableNumber.ReplaceAllString(message, placeholderNumber)

	// Collapse whitespace last, once every placeholder above is in place
	return strings.TrimSpace(repeatedSpace.ReplaceAllString(message, " "))
}
