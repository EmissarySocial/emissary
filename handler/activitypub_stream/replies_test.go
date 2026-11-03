package activitypub_stream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// TestGetRepliesCollection_ClosedIsEmpty pins that a Stream with closed replies serves an empty Replies
// collection without loading it, so the replies it already holds stay hidden but kept (REPLY-CONTROLS §1.2)
func TestGetRepliesCollection_ClosedIsEmpty(t *testing.T) {

	stream := model.NewStream()
	stream.URL = "https://local.example/" + stream.StreamID.Hex()
	stream.DefaultAllow = model.Permissions{model.MagicGroupIDAnonymous}
	stream.ReplyPolicy = model.ReplyPolicyNobody

	request := httptest.NewRequest(http.MethodGet, "/"+stream.StreamID.Hex()+"/pub/replies", nil)
	recorder := httptest.NewRecorder()
	ctx := &steranko.Context{Context: echo.New().NewContext(request, recorder)}
	actorID := ""

	// A nil factory and session prove that the closed path reads nothing
	require.NoError(t, GetRepliesCollection(ctx, nil, nil, &actorID, &stream))
	require.Equal(t, http.StatusOK, recorder.Code)

	body := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, stream.ActivityPubRepliesURL(), body["id"])
	require.EqualValues(t, 0, body["totalItems"])
}
