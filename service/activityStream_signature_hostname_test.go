package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestActivityStream_RequestHostname verifies that the service names the host the server
// factory reads, and names none before it is wired
func TestActivityStream_RequestHostname(t *testing.T) {

	request := httptest.NewRequest(http.MethodPost, "https://internal.example/inbox", nil)

	unwired := NewActivityStream()
	require.Empty(t, unwired.RequestHostname(request))

	factory := &Factory{serverFactory: &lifecycleServerFactory{}}

	wired := NewActivityStream()
	wired.requestHostname = factory.RequestHostname
	require.Equal(t, "internal.example", wired.RequestHostname(request))
}
