package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsReadOnlyMethod verifies that HEAD shares GET's read-only session, and every
// state-changing method still gets a transaction.
func TestIsReadOnlyMethod(t *testing.T) {

	require.True(t, isReadOnlyMethod(http.MethodGet))
	require.True(t, isReadOnlyMethod(http.MethodHead))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		require.False(t, isReadOnlyMethod(method), method)
	}
}
