package domain

import (
	"context"
	"net/http"

	"github.com/benpate/ghost/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HTTPRequest wraps an HTTP request and tracks the properties of it that have been accessed.  This is used
// to construct an accurate "Vary" header for cached responses.
type HTTPRequest struct {
	request *http.Request
	used    map[string]bool
}

// NewHTTPRequest wraps an http.Request object
func NewHTTPRequest(original *http.Request) *HTTPRequest {
	return &HTTPRequest{request: original}
}

// Context returns the golang context for this http request
func (r *HTTPRequest) Context() context.Context {
	return r.request.Context()
}

// QueryParam returns a query parameter from the requested URL
func (r HTTPRequest) QueryParam(name string) string {
	return r.request.URL.Query().Get(name)
}

// View returns the URL query parameter "view" from the HTTP request
func (r HTTPRequest) View() string {
	return r.QueryParam("view")
}

// Transition returns the URL query parameter "transition" from the HTTP request
func (r HTTPRequest) Transition() string {
	return r.QueryParam("transition")
}

// UserID returns the userID of the current user
func (r *HTTPRequest) UserID() primitive.ObjectID {
	return primitive.NewObjectID()
}

// TemplateID returns the requested templateId from the URL query parameter
func (r *HTTPRequest) TemplateID() string {
	return r.QueryParam("templateId")
}

// ParentToken returns the requested parentId from the URL query parameter
func (r *HTTPRequest) ParentToken() string {
	return r.QueryParam("parent")
}

// Groups returns a sorted slice of strings containing the group names that the user behind this request belongs to.
func (r *HTTPRequest) Groups() []string {
	return []string{}
}

// Partial returns TRUE if this is a request for a partial page (HTML fragment), and not a complete HTML page.
func (r *HTTPRequest) Partial() bool {
	return (r.request.Header.Get("HX-Request") != "")
}

func (r *HTTPRequest) objectID(value string) primitive.ObjectID {

	if result, err := primitive.ObjectIDFromHex(value); err == nil {
		return result
	}

	return service.ZeroObjectID()
}
