package cimd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestGetMetadata verifies each answer a client_id can give: a matching document, a fallback,
// or a refusal, with the request asking for JSON
func TestGetMetadata(t *testing.T) {

	// The server answers each path with its own status, content type, and body
	type answer struct {
		status      int
		contentType string
		body        string
	}

	answers := map[string]answer{}
	accepted := make(chan string, 100)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accepted <- r.Header.Get("Accept")
		answer := answers[r.URL.Path]
		w.Header().Set("Content-Type", answer.contentType)
		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	t.Cleanup(server.Close)

	// The server listens on 127.0.0.1, and a client_id must name a domain
	base := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)

	serve := func(path string, status int, contentType string, body string) string {
		answers[path] = answer{status: status, contentType: contentType, body: body}
		return base + path
	}

	t.Run("a matching document is used", func(t *testing.T) {
		clientID := serve("/client.json", http.StatusOK, "application/json", "")
		answers["/client.json"] = answer{http.StatusOK, "application/json", `{"client_id":"` + clientID + `","client_name":"Example App","redirect_uris":["` + base + `/callback"]}`}

		metadata, published, err := GetMetadata("localhost", clientID, true)
		require.NoError(t, err)
		require.True(t, published)
		require.Equal(t, "Example App", metadata.ClientName)
		require.Equal(t, []string{base + "/callback"}, []string(metadata.RedirectURIs))
		require.Equal(t, "application/json", <-accepted)
	})

	t.Run("a document naming the client in another case still matches", func(t *testing.T) {
		clientID := serve("/case.json", http.StatusOK, "application/json", "")
		answers["/case.json"] = answer{http.StatusOK, "application/json", `{"client_id":"` + strings.Replace(clientID, "http://localhost", "HTTP://LOCALHOST", 1) + `"}`}

		_, published, err := GetMetadata("localhost", clientID, true)
		require.NoError(t, err)
		require.True(t, published)
		<-accepted
	})

	t.Run("a document naming another client is refused", func(t *testing.T) {
		clientID := serve("/impostor.json", http.StatusOK, "application/json", `{"client_id":"https://real-app.example.com/client.json"}`)

		_, published, err := GetMetadata("localhost", clientID, true)
		require.True(t, derp.IsBadRequest(err), "answers %d", derp.ErrorCode(err))
		require.False(t, published)
		secretcheck.RequireAbsent(t, err, "real-app")
		<-accepted
	})

	// Each of these answers publishes no metadata, so the client is described from its URL
	fallback := func(name string, path string, status int, contentType string, body string) {
		t.Run(name, func(t *testing.T) {
			clientID := serve(path, status, contentType, body)
			metadata, published, err := GetMetadata("localhost", clientID, true)
			require.NoError(t, err)
			require.False(t, published)
			require.Equal(t, Metadata{}, metadata)
			<-accepted
		})
	}

	fallback("an HTML page", "/", http.StatusOK, "text/html", `<html><head><title>Bridgy Fed</title></head></html>`)
	fallback("JSON with no client_id", "/actor", http.StatusOK, "application/activity+json", `{"id":"https://example.com/actor","type":"Application"}`)
	fallback("JSON that is not an object", "/list", http.StatusOK, "application/json", `["a","b"]`)
	fallback("a body that is not JSON", "/broken", http.StatusOK, "application/json", `{"client_id":`)
	fallback("an empty body", "/empty", http.StatusOK, "application/json", ``)

	// Each of these answers names no client that exists, so the lookup fails and no client is created
	refusal := func(name string, path string, status int, contentType string, body string) {
		t.Run(name, func(t *testing.T) {
			clientID := serve(path, status, contentType, body)
			_, published, err := GetMetadata("localhost", clientID, true)
			require.True(t, derp.IsBadRequest(err), "answers %d", derp.ErrorCode(err))
			require.False(t, published)
			secretcheck.RequireAbsent(t, err, "secret-looking")
			<-accepted
		})
	}

	refusal("a 404", "/missing", http.StatusNotFound, "text/html", `secret-looking body`)
	refusal("a 410", "/gone", http.StatusGone, "text/plain", `Gone`)
	refusal("a 304", "/unchanged", http.StatusNotModified, "text/plain", ``)
	refusal("a body over the size limit", "/huge", http.StatusOK, "application/json", `{"client_name":"`+strings.Repeat("x", maxDocumentSize)+`"}`)

	t.Run("a server error is refused, so the next attempt retries", func(t *testing.T) {
		clientID := serve("/down", http.StatusServiceUnavailable, "text/plain", "secret-looking body")

		_, published, err := GetMetadata("localhost", clientID, true)
		require.True(t, derp.IsBadGateway(err), "answers %d", derp.ErrorCode(err))
		require.False(t, published)
		secretcheck.RequireAbsent(t, err, "secret-looking")
		<-accepted
	})
}

// TestGetMetadata_Refusals verifies the client_ids that are refused before, or instead of, a request
func TestGetMetadata_Refusals(t *testing.T) {

	// A client_id that breaks the IndieAuth rules is never fetched
	_, _, err := GetMetadata("localhost", "https://127.0.0.1/client.json", true)
	require.True(t, derp.IsBadRequest(err))

	// A production host does not fetch a local client
	_, _, err = GetMetadata("emissary.example.com", "http://localhost:8080/client.json", true)
	require.True(t, derp.IsBadRequest(err))

	// A client that cannot be reached is refused as a bad gateway
	server := httptest.NewServer(http.NotFoundHandler())
	clientID := strings.Replace(server.URL, "127.0.0.1", "localhost", 1) + "/client.json"
	server.Close()

	_, published, err := GetMetadata("localhost", clientID, true)
	require.True(t, derp.IsBadGateway(err), "answers %d", derp.ErrorCode(err))
	require.False(t, published)
}

// TestGetMetadata_PrivateIPs verifies that the private-IP policy reaches the request: a domain that may
// not fetch private addresses cannot reach a client on one, even a local client allowed on a local host
func TestGetMetadata_PrivateIPs(t *testing.T) {

	requested := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requested = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	clientID := strings.Replace(server.URL, "127.0.0.1", "localhost", 1) + "/client.json"

	_, published, err := GetMetadata("localhost", clientID, false)
	require.Error(t, err)
	require.False(t, published)
	require.False(t, requested, "a private address was fetched")
}

// TestGetMetadata_Redirects verifies that a redirect is followed to a page, and never to a document
func TestGetMetadata_Redirects(t *testing.T) {

	var base string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch r.URL.Path {

		case "/to-page":
			http.Redirect(w, r, base+"/page", http.StatusFound)

		case "/to-document":
			http.Redirect(w, r, base+"/document", http.StatusFound)

		case "/to-nowhere":
			http.Redirect(w, r, base+"/missing", http.StatusFound)

		case "/page":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html></html>`))

		case "/document":
			// The document names the address that redirected here, which is the impersonation to refuse
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"client_id":"` + base + `/to-document","client_name":"Borrowed","redirect_uris":["https://evil.example.com/callback"]}`))

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	base = strings.Replace(server.URL, "127.0.0.1", "localhost", 1)

	// A redirect to an HTML page is an IndieAuth client that moved its home page
	metadata, published, err := GetMetadata("localhost", base+"/to-page", true)
	require.NoError(t, err)
	require.False(t, published)
	require.Equal(t, Metadata{}, metadata)

	// A redirect to a document is refused, even when the document names the address that was asked for
	_, published, err = GetMetadata("localhost", base+"/to-document", true)
	require.True(t, derp.IsBadRequest(err), "answers %d", derp.ErrorCode(err))
	require.False(t, published)

	// A redirect that ends in a 404 names nothing
	_, published, err = GetMetadata("localhost", base+"/to-nowhere", true)
	require.True(t, derp.IsBadRequest(err), "answers %d", derp.ErrorCode(err))
	require.False(t, published)

}
