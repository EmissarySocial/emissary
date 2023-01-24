package render

import (
	"html/template"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Renderer safely wraps model objects for consumption by an html Template
type Renderer interface {

	// Render is the main entry-point for templates to use a Renderer
	Render() (template.HTML, error)     // Render function outputs an HTML template
	View(string) (template.HTML, error) // Render function outputs an HTML template

	// COMMON API METHODS
	Host() string                        // String representation of the protocol + hostname
	Protocol() string                    // String representation of the HTTP protocol to use when addressing this record (http:// or https://)
	Hostname() string                    // Hostname for this server
	Token() string                       // URL Token of the record being rendered
	TopLevelID() string                  // ID of the Top-Level item to highlight in the navigation.
	PageTitle() string                   // Human-friendly title to put at the top of the page.
	Summary() string                     // Human-friendly summary to put at the top of the page (maybe)
	Permalink() string                   // Permanent link to the stream being rendered
	URL() string                         // Complete URL of the requested page
	QueryParam(string) string            // Query parameter of the requested page
	ActionID() string                    // Token that identifies the action requested via the URL.
	Action() *model.Action               // The pipeline action to be taken by this renderer
	IsAuthenticated() bool               // Returns TRUE if the user is signed in
	IsPartialRequest() bool              // Returns TRUE if this is an HTMX request for a page fragment
	UseGlobalWrapper() bool              // Returns TRUE if this renderer uses the common site chrome.
	UserCan(string) bool                 // Returns TRUE if the signed-in user has access to the named action
	AuthenticatedID() primitive.ObjectID // Returns the ID of the signed-in user (or zero if not signed in)

	GetBool(name string) bool
	GetFloat(name string) float64
	GetInt(name string) int
	GetInt64(name string) int64
	GetString(name string) string
	setBool(name string, value bool)
	setFloat(name string, value float64)
	setInt(name string, value int)
	setInt64(name string, value int64)
	setString(name string, value string)

	factory() Factory                    // The service factory
	context() *steranko.Context          // The request context embedded in the Renderer
	service() service.ModelService       // The abstracted ModelService the backs this Renderer
	templateRole() string                // Returns the role that the current template plays in the system. Used for choosing child template.
	template() *model.Template           // The template used for this renderer (if any)
	objectType() string                  // The type of object being rendered
	authorization() model.Authorization  // retrieves the user's authorization data from the context
	schema() schema.Schema               // Schema to use to validate this Object
	object() data.Object                 // Model Object being rendered
	objectID() primitive.ObjectID        // MongoDB ObjectID of the Object being rendered
	setQuery(string, string)             // Sets a queryString parameter
	getUser() (model.User, error)        // Retrieves the currently-logged-in user
	lookupProvider() form.LookupProvider // Retrieves the LookupProvider for this user

	executeTemplate(io.Writer, string, any) error // The HTML template used by this Renderer
}

// TODO: LOW: This is expensive and abstract.  Minimize the use of this function as much as possible.
// This function is only used in one place, so perhaps we can just inline it?
func NewRenderer(factory Factory, ctx *steranko.Context, object data.Object, actionID string) (Renderer, error) {

	switch object := object.(type) {

	case *model.Stream:
		return NewStreamWithoutTemplate(factory, ctx, object, actionID)

	case *model.User:
		return NewProfile(factory, ctx, object, actionID)
	}

	spew.Dump(object)
	return nil, derp.NewInternalError("render.NewRenderer", "Unrecognized object", object)
}
