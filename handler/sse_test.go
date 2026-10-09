package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/realtime"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestServerSentEvent_HeadReturnsHeadersWithoutStreaming verifies that HEAD answers with the
// event-stream headers and returns, instead of registering a broker client and holding the connection.
func TestServerSentEvent_HeadReturnsHeadersWithoutStreaming(t *testing.T) {

	request := httptest.NewRequest(http.MethodHead, "https://example.com/albums/sse", nil)
	recorder := httptest.NewRecorder()
	ctx := &steranko.Context{Context: echo.New().NewContext(request, recorder)}

	// A nil Factory proves the broker is never reached: dereferencing it would panic
	err := serverSentEvent(ctx, nil, primitive.NewObjectID(), realtime.TopicUpdated)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, model.MimeTypeEventStream, recorder.Header().Get("Content-Type"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Empty(t, recorder.Body.String())
}
