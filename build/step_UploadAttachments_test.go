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

	reader, contentType, err := verifyContentType(source, "photo.jpg", "image/*")

	require.Error(t, err)
	require.Nil(t, reader)
	require.Zero(t, contentType)
}

// TestVerifyContentType_AcceptImage confirms that a real image is accepted AND
// that the returned reader replays the full, unmodified byte stream.
func TestVerifyContentType_AcceptImage(t *testing.T) {

	// Minimal valid PNG header + body (enough for http.DetectContentType to sniff "image/png").
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("payload-bytes", 100))

	reader, contentType, err := verifyContentType(bytes.NewReader(png), "photo.png", "image/*")

	require.Nil(t, err)
	require.NotNil(t, reader)
	require.Equal(t, "image/png", contentType)

	// The reader must replay every original byte, in order, with nothing lost or duplicated.
	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, png, roundTrip)
}

// TestVerifyContentType_EmptyAcceptAllows confirms legacy behavior: when no accept
// pattern is configured, any file is accepted and the stream is replayed intact.
// The content-type is still detected, because it is recorded on the Attachment.
func TestVerifyContentType_EmptyAcceptAllows(t *testing.T) {

	payload := []byte("arbitrary non-image bytes")

	reader, contentType, err := verifyContentType(bytes.NewReader(payload), "notes.txt", "")

	require.Nil(t, err)
	require.NotNil(t, reader)
	require.Equal(t, "text/plain", contentType)

	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, payload, roundTrip)
}

// TestVerifyContentType_Polyglot confirms that a file whose bytes pass the accept
// pattern is reported by its *sniffed* type, not the type its filename implies.
// An HTML document behind GIF magic bytes is "image/gif" here -- model.Attachment
// is what refuses to serve it inline.
func TestVerifyContentType_Polyglot(t *testing.T) {

	polyglot := []byte("GIF89a=1;\n<script>window.__xss=document.domain</script>")

	reader, contentType, err := verifyContentType(bytes.NewReader(polyglot), "poc.html", "image/*")

	require.Nil(t, err)
	require.NotNil(t, reader)
	require.Equal(t, "image/gif", contentType)
}

// TestVerifyContentType_AcceptFLAC confirms that a FLAC upload passes an "audio/*"
// accept pattern -- the file that started it all was rejected because the standard
// library cannot sniff FLAC.
func TestVerifyContentType_AcceptFLAC(t *testing.T) {

	flac := append([]byte("fLaC\x00\x00\x00\x22\x10\x00\x10\x00"), bytes.Repeat([]byte{0xA5}, 600)...)

	reader, contentType, err := verifyContentType(bytes.NewReader(flac), "02 Key 13 - Primavera 2024.flac", "audio/*")

	require.Nil(t, err)
	require.NotNil(t, reader)
	require.Equal(t, "audio/flac", contentType)

	// The reader must replay every original byte, in order, with nothing lost or duplicated.
	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, flac, roundTrip)
}

// TestVerifyContentType_ShortFile confirms that a file smaller than the 512-byte
// sniff window is handled without error and replayed intact.
func TestVerifyContentType_ShortFile(t *testing.T) {

	payload := []byte("tiny")

	reader, _, err := verifyContentType(bytes.NewReader(payload), "tiny.txt", "")

	require.Nil(t, err)

	roundTrip, err := io.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, payload, roundTrip)
}
