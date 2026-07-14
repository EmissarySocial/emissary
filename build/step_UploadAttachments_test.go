package build

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentTypeMatches(t *testing.T) {

	table := []struct {
		detected   string
		acceptType string
		expected   bool
	}{
		// Wildcard category matches
		{"image/png", "image/*", true},
		{"image/jpeg", "image/*", true},
		{"image/webp", "image/*", true},
		{"text/plain", "image/*", false},
		{"application/octet-stream", "image/*", false},
		{"audio/mpeg", "image/*", false},

		// "*/*" and empty patterns accept anything
		{"application/octet-stream", "*/*", true},
		{"text/plain", "", true},

		// Exact media-type matches
		{"image/png", "image/png", true},
		{"image/png", "image/jpeg", false},

		// Comma-separated lists (any match wins)
		{"image/webp", "image/png,image/webp", true},
		{"image/gif", "image/png,image/webp", false},
		{"audio/mpeg", "image/*,audio/*", true},

		// DetectContentType often appends "; charset=..." -- the media type still matches
		{"text/plain; charset=utf-8", "text/*", true},
		{"text/plain; charset=utf-8", "image/*", false},

		// Whitespace in the pattern list is tolerated
		{"image/png", " image/* , audio/* ", true},
	}

	for _, row := range table {
		result := contentTypeMatches(row.detected, row.acceptType)
		require.Equal(t, row.expected, result, "detected=%q accept=%q", row.detected, row.acceptType)
	}
}

// TestVerifyContentType_Reject confirms that a non-image payload (a text file
// masquerading as a .jpg) is rejected when the step only accepts images.
func TestVerifyContentType_Reject(t *testing.T) {

	source := strings.NewReader("this is definitely not an image, it is plain text")

	reader, err := verifyContentType(source, "image/*")

	require.Error(t, err)
	require.Nil(t, reader)
}

// TestVerifyContentType_AcceptImage confirms that a real image is accepted AND
// that the returned reader replays the full, unmodified byte stream.
func TestVerifyContentType_AcceptImage(t *testing.T) {

	// Minimal valid PNG header + body (enough for http.DetectContentType to sniff "image/png").
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("payload-bytes", 100))

	reader, err := verifyContentType(bytes.NewReader(png), "image/*")

	require.Nil(t, err)
	require.NotNil(t, reader)

	// The reader must replay every original byte, in order, with nothing lost or duplicated.
	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, png, roundTrip)
}

// TestVerifyContentType_EmptyAcceptAllows confirms legacy behavior: when no accept
// pattern is configured, any file is accepted and the stream is replayed intact.
func TestVerifyContentType_EmptyAcceptAllows(t *testing.T) {

	payload := []byte("arbitrary non-image bytes")

	reader, err := verifyContentType(bytes.NewReader(payload), "")

	require.Nil(t, err)
	require.NotNil(t, reader)

	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, payload, roundTrip)
}

// TestVerifyContentType_ShortFile confirms that a file smaller than the 512-byte
// sniff window is handled without error and replayed intact.
func TestVerifyContentType_ShortFile(t *testing.T) {

	payload := []byte("tiny")

	reader, err := verifyContentType(bytes.NewReader(payload), "")

	require.Nil(t, err)

	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, payload, roundTrip)
}
