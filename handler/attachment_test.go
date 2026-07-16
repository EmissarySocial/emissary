package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// invokeSetAttachmentHeaders runs one Attachment through setAttachmentHeaders and returns
// the response headers a visitor would receive.
func invokeSetAttachmentHeaders(t *testing.T, original string, contentType string, requestPath string) http.Header {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, requestPath, nil)
	recorder := httptest.NewRecorder()
	ctx := &steranko.Context{Context: echo.New().NewContext(request, recorder)}

	attachment := model.NewAttachment(model.AttachmentObjectTypeStream, primitive.NewObjectID())
	attachment.Original = original
	attachment.ContentType = contentType

	// The FileSpec must be the one MediaServer.Serve would be handed, because the
	// headers describe the file that it generates.
	setAttachmentHeaders(ctx, attachment, attachment.FileSpec(request.URL))

	return recorder.Header()
}

// TestSetAttachmentHeaders_Image confirms that re-encoded media is still rendered inline.
func TestSetAttachmentHeaders_Image(t *testing.T) {

	header := invokeSetAttachmentHeaders(t, "photo.png", "image/png", "/abc123.webp")

	// MediaServer re-encodes this file, so it is typed as the format it will generate.
	require.Equal(t, "image/webp", header.Get("Content-Type"))
	require.Equal(t, "nosniff", header.Get("X-Content-Type-Options"))

	// An image must NOT be forced into a download, or every post breaks.
	require.Zero(t, header.Get("Content-Disposition"))
}

// TestSetAttachmentHeaders_HTML is the regression test for the reported stored XSS: an
// uploaded HTML file, requested with a .html extension, must not come back as a document.
func TestSetAttachmentHeaders_HTML(t *testing.T) {

	header := invokeSetAttachmentHeaders(t, "poc.html", "text/html", "/abc123.html")

	require.Equal(t, "application/octet-stream", header.Get("Content-Type"))
	require.Equal(t, "attachment; filename=poc.html", header.Get("Content-Disposition"))
	require.Equal(t, "nosniff", header.Get("X-Content-Type-Options"))
	require.Equal(t, "default-src 'none'; sandbox", header.Get("Content-Security-Policy"))

	// The bug itself: http.ServeContent would have typed this from the *request* URL.
	require.NotContains(t, header.Get("Content-Type"), "text/html")
}

// TestSetAttachmentHeaders_Polyglot confirms that a file which passes an "image/*"
// accept-type on upload, but which MediaServer copies verbatim, is still a download.
func TestSetAttachmentHeaders_Polyglot(t *testing.T) {

	header := invokeSetAttachmentHeaders(t, "poc.html", "image/gif", "/abc123.html")

	require.Equal(t, "application/octet-stream", header.Get("Content-Type"))
	require.Equal(t, "attachment; filename=poc.html", header.Get("Content-Disposition"))
}

// TestSetAttachmentHeaders_NoExtension confirms that stripping the extension from the URL
// does not reopen the hole -- http.ServeContent would sniff the bytes and find HTML.
func TestSetAttachmentHeaders_NoExtension(t *testing.T) {

	header := invokeSetAttachmentHeaders(t, "poc.html", "text/html", "/abc123")

	require.Equal(t, "application/octet-stream", header.Get("Content-Type"))
	require.Equal(t, "attachment; filename=poc.html", header.Get("Content-Disposition"))
}

// TestSetAttachmentHeaders_Legacy confirms that Attachments uploaded before content
// sniffing keep working: the filename alone decides, as it always did for them.
func TestSetAttachmentHeaders_Legacy(t *testing.T) {

	image := invokeSetAttachmentHeaders(t, "legacy.png", "", "/abc123.webp")
	require.Equal(t, "image/webp", image.Get("Content-Type"))
	require.Zero(t, image.Get("Content-Disposition"))

	document := invokeSetAttachmentHeaders(t, "legacy.html", "", "/abc123.html")
	require.Equal(t, "application/octet-stream", document.Get("Content-Type"))
}

// TestAttachmentContentDisposition confirms that an attacker-chosen filename cannot
// escape the header it is written into.
func TestAttachmentContentDisposition(t *testing.T) {

	table := []struct {
		filename string
		expected string
	}{
		{"report.pdf", "attachment; filename=report.pdf"},

		// A quoted filename is escaped, not closed early.
		{`in"jected.pdf`, `attachment; filename="in\"jected.pdf"`},

		// CRLF is percent-encoded, so it cannot forge a second header.
		{"evil\r\nX-Injected: yes", "attachment; filename*=utf-8''evil%0D%0AX-Injected%3A%20yes"},

		// Non-ASCII names are encoded per RFC 2231 rather than dropped.
		{"café.pdf", "attachment; filename*=utf-8''caf%C3%A9.pdf"},

		// A missing name still forces a download.
		{"", "attachment"},
		{"   ", "attachment"},
	}

	for _, row := range table {
		require.Equal(t, row.expected, attachmentContentDisposition(row.filename), "filename=%q", row.filename)
	}
}
