package domain

import (
	"context"
	"net/http"

	"github.com/benpate/ghost/model"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HTTPRequest wraps an HTTP request and tracks the properties of it that have been accessed.  This is used
// to construct an accurate "Vary" header for cached responses.
type HTTPRequest struct {
	request       *http.Request
	authorization *model.Authorization
}

// NewHTTPRequest wraps an http.Request object
func NewHTTPRequest(ctx echo.Context) *HTTPRequest {

	result := HTTPRequest{
		request: ctx.Request(),
	}

	return &result
}

func (r *HTTPRequest) URL() string {
	return r.request.RequestURI
}

// Context returns the golang context for this http request
func (r *HTTPRequest) Context() context.Context {
	return r.request.Context()
}

// QueryParam returns a query parameter from the requested URL
func (r HTTPRequest) QueryParam(name string) string {
	return r.request.URL.Query().Get(name)
}

// UserID returns the userID of the current user
func (r *HTTPRequest) UserID() primitive.ObjectID {
	return primitive.NewObjectID()
}

// Partial returns TRUE if this is a request for a partial page (HTML fragment), and not a complete HTML page.
func (r *HTTPRequest) Partial() bool {
	return (r.request.Header.Get("HX-Request") != "")
}

func (r *HTTPRequest) Authorization() *model.Authorization {
	return r.authorization
}
