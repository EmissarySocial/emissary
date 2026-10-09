package content

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// newFileServer returns the address of a test server that answers every request with handler.
// RULE: The adapter is built with allowPrivateIPs TRUE, because httptest listens on 127.0.0.1.
func newFileServer(t *testing.T, handler http.HandlerFunc) (HTTPS, string) {

	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewHTTPS(true), server.URL + "/docs/attachments/file.png"
}

// readAll reads a FetchFile result to the end, and closes it
func readAll(t *testing.T, reader io.ReadCloser) ([]byte, error) {

	t.Helper()

	defer func() { require.NoError(t, reader.Close()) }()

	return io.ReadAll(reader)
}

// TestHTTPS_FetchFile returns the file's bytes, whatever type the server declares
func TestHTTPS_FetchFile(t *testing.T) {

	adapter, address := newFileServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("PNG-BYTES"))
	})

	reader, err := adapter.FetchFile(context.Background(), address, 100)
	require.NoError(t, err)

	body, err := readAll(t, reader)
	require.NoError(t, err)
	require.Equal(t, "PNG-BYTES", string(body))
}

// TestHTTPS_FetchFile_AtTheCap keeps a file exactly as large as the cap
func TestHTTPS_FetchFile_AtTheCap(t *testing.T) {

	adapter, address := newFileServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 64)))
	})

	reader, err := adapter.FetchFile(context.Background(), address, 64)
	require.NoError(t, err)

	body, err := readAll(t, reader)
	require.NoError(t, err)
	require.Len(t, body, 64)
}

// TestHTTPS_FetchFile_DeclaredTooLarge refuses a file whose declared length is over the cap,
// before reading any of it
func TestHTTPS_FetchFile_DeclaredTooLarge(t *testing.T) {

	adapter, address := newFileServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(65))
		_, _ = w.Write([]byte(strings.Repeat("x", 65)))
	})

	_, err := adapter.FetchFile(context.Background(), address, 64)
	require.Error(t, err)
	require.True(t, derp.IsClientError(err), "an oversize file is the author's to fix")
}

// TestHTTPS_FetchFile_UndeclaredTooLarge fails the READ of a body that declared no length and
// then ran past the cap.  Stopping quietly would store a truncated file.
func TestHTTPS_FetchFile_UndeclaredTooLarge(t *testing.T) {

	adapter, address := newFileServer(t, func(w http.ResponseWriter, _ *http.Request) {

		// Flushing before the body is complete forces chunked encoding, so no length is declared
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte(strings.Repeat("x", 40)))
		flusher.Flush()
		_, _ = w.Write([]byte(strings.Repeat("x", 40)))
	})

	reader, err := adapter.FetchFile(context.Background(), address, 64)
	require.NoError(t, err, "nothing is known to be wrong until the body is read")

	_, err = readAll(t, reader)
	require.Error(t, err)
	require.True(t, derp.IsClientError(err))
}

// TestHTTPS_FetchFile_StatusCodes classifies failures the same way the source itself is classified
func TestHTTPS_FetchFile_StatusCodes(t *testing.T) {

	tests := map[int]func(error) bool{
		http.StatusNotFound:            derp.IsNotFound,
		http.StatusForbidden:           derp.IsClientError,
		http.StatusInternalServerError: derp.IsServerError,
	}

	for status, isExpected := range tests {

		adapter, address := newFileServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})

		_, err := adapter.FetchFile(context.Background(), address, 64)
		require.True(t, isExpected(err), "status %d", status)
	}
}

// TestHTTPS_FetchFile_TooManyRequests keeps a 429 as a 429, so the queue waits instead of failing
func TestHTTPS_FetchFile_TooManyRequests(t *testing.T) {

	adapter, address := newFileServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := adapter.FetchFile(context.Background(), address, 64)

	isTooMany, _ := derp.IsTooManyRequests(err)
	require.True(t, isTooMany)
}

// TestHTTPS_FetchFile_AddressRules applies the same address rules as the source itself
func TestHTTPS_FetchFile_AddressRules(t *testing.T) {

	_, err := NewHTTPS(false).FetchFile(context.Background(), "http://example.com/attachments/a.png", 64)
	require.True(t, derp.IsClientError(err), "plain http is refused where private addresses are")

	_, err = NewHTTPS(false).FetchFile(context.Background(), "https://user:secret@example.com/attachments/a.png", 64)
	require.True(t, derp.IsClientError(err))
	require.NotContains(t, derp.Message(err), "secret")
}
