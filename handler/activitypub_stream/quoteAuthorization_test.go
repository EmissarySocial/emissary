package activitypub_stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// getQuoteAuthorization routes one request through the real path, and returns what the handler answered
// and how many reads the session saw
func getQuoteAuthorization(t *testing.T, stream *model.Stream, quoteAuthorizationID string) (*httptest.ResponseRecorder, int) {

	t.Helper()

	// A zero Factory is enough: each case below answers before any service reads a field
	factory := &service.Factory{}
	session := &countingSession{}

	request := httptest.NewRequest(http.MethodGet, "/"+stream.StreamID.Hex()+"/pub/quoteAuthorization/"+quoteAuthorizationID, nil)
	recorder := httptest.NewRecorder()

	router := echo.New()
	router.GET("/:stream/pub/quoteAuthorization/:quoteAuthorizationId", func(ctx echo.Context) error {
		return GetQuoteAuthorization(&steranko.Context{Context: ctx}, factory, session, stream)
	})

	router.ServeHTTP(recorder, request)
	return recorder, session.reads
}

// TestGetQuoteAuthorization_InvalidID pins that a malformed QuoteAuthorization ID answers 404 directly,
// without reading anything and without returning an error for the error log
func TestGetQuoteAuthorization_InvalidID(t *testing.T) {

	stream := model.NewStream()
	stream.DefaultAllow = model.Permissions{model.MagicGroupIDAnonymous}

	for _, value := range []string{"not-an-id", "12345", "zzzzzzzzzzzzzzzzzzzzzzzz", "1234567812345678123456781"} {
		recorder, reads := getQuoteAuthorization(t, &stream, value)
		require.Equal(t, http.StatusNotFound, recorder.Code, value)
		require.Empty(t, recorder.Body.String(), value)
		require.Zero(t, reads, value)
	}
}

// TestGetQuoteAuthorization_Closed pins that a Stream which no longer accepts quotes answers 404 for
// every QuoteAuthorization it granted, without reading them
func TestGetQuoteAuthorization_Closed(t *testing.T) {

	closed := model.NewStream()
	closed.DefaultAllow = model.Permissions{model.MagicGroupIDAnonymous}
	closed.ReplyPolicy = model.ReplyPolicyNobody

	private := model.NewStream()
	private.DefaultAllow = model.Permissions{model.MagicGroupIDAuthenticated}

	for name, stream := range map[string]*model.Stream{"closed": &closed, "private": &private} {
		recorder, reads := getQuoteAuthorization(t, stream, primitive.NewObjectID().Hex())
		require.Equal(t, http.StatusNotFound, recorder.Code, name)
		require.Empty(t, recorder.Body.String(), name)
		require.Zero(t, reads, name)
	}
}

/******************************************
 * countingSession counts every collection
 * it hands out, and finds nothing in any
 ******************************************/

// countingSession is a data.Session whose collections are all empty
type countingSession struct {
	reads int
}

// Collection counts the request and returns an empty collection
func (session *countingSession) Collection(string) data.Collection {
	session.reads++
	return emptyCollection{}
}

// Context implements the data.Session interface
func (session *countingSession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface
func (session *countingSession) Close() {}

// emptyCollection is a data.Collection that holds nothing
type emptyCollection struct{}

// Context implements the data.Collection interface
func (emptyCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface
func (emptyCollection) Count(exp.Expression, ...option.Option) (int64, error) { return 0, nil }

// Query implements the data.Collection interface
func (emptyCollection) Query(any, exp.Expression, ...option.Option) error { return nil }

// Iterator implements the data.Collection interface
func (emptyCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "emptyCollection holds nothing")
}

// Load implements the data.Collection interface
func (emptyCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "emptyCollection holds nothing")
}

// Save implements the data.Collection interface
func (emptyCollection) Save(data.Object, string) error { return nil }

// Delete implements the data.Collection interface
func (emptyCollection) Delete(data.Object, string) error { return nil }

// HardDelete implements the data.Collection interface
func (emptyCollection) HardDelete(exp.Expression) error { return nil }
