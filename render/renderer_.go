package render

import (
	"html/template"
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
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
	NavigationID() string                // ID of the Top-Level item to highlight in the navigation.
	PageTitle() string                   // Human-friendly title to put at the top of the page.
	Summary() string                     // Human-friendly summary to put at the top of the page (maybe)
	Permalink() string                   // Permanent link to the record being rendered
	BasePath() string                    // URL Path of the root of this object, without any additional actions.
	URL() string                         // Complete URL of the requested page
	QueryParam(string) string            // Query parameter of the requested page
	SetQueryParam(string, string)        // Sets a queryString parameter
	ActionID() string                    // Token that identifies the action requested via the URL.
	Action() model.Action                // The pipeline action to be taken by this renderer
	IsAuthenticated() bool               // Returns TRUE if the user is signed in
	IsPartialRequest() bool              // Returns TRUE if this is an HTMX request for a page fragment
	UserCan(string) bool                 // Returns TRUE if the signed-in user has access to the named action
	AuthenticatedID() primitive.ObjectID // Returns the ID of the signed-in user (or zero if not signed in)

	getArguments() map[string]string // Returns the arguments passed to the action
	GetBool(name string) bool
	GetFloat(name string) float64
	GetHTML(name string) template.HTML
	GetInt(name string) int
	GetInt64(name string) int64
	GetString(name string) string
	setString(name string, value string)

	GetContent() template.HTML
	SetContent(string)

	factory() Factory                    // The service factory
	request() *http.Request              // The original http.Request that we are responding to
	response() http.ResponseWriter       // The original http.ResponseWriter that we are responding to
	authorization() model.Authorization  // The user's authorization data from the context
	service() service.ModelService       // The abstracted ModelService the backs this Renderer
	templateRole() string                // Returns the role that the current template plays in the system. Used for choosing child template.
	template() model.Template            // The template used for this renderer (if any)
	objectType() string                  // The type of object being rendered
	schema() schema.Schema               // Schema to use to validate this Object
	object() data.Object                 // Model Object being rendered
	objectID() primitive.ObjectID        // MongoDB ObjectID of the Object being rendered
	getUser() (model.User, error)        // Retrieves the currently-logged-in user
	lookupProvider() form.LookupProvider // Retrieves the LookupProvider for this user
	debug()                              // Outputs debug information to the console
	clone(string) (Renderer, error)      // Creates a new Renderer with the same type and object, but a different action

	executeTemplate(io.Writer, string, any) error // The HTML template used by this Renderer
}
