package build

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

// Builder safely wraps model objects for consumption by an html Template
type Builder interface {

	// Render is the main entry-point for templates to use a Builder
	Render() (template.HTML, error)     // Render function outputs an HTML template
	View(string) (template.HTML, error) // Render function outputs an HTML template

	// COMMON API METHODS
	Protocol() string                    // String representation of the HTTP protocol to use when addressing this record (http:// or https://)
	Host() string                        // String representation of the protocol + hostname
	Hostname() string                    // Hostname for this server
	Token() string                       // URL Token of the record being built
	NavigationID() string                // ID of the Top-Level item to highlight in the navigation.
	PageTitle() string                   // Human-friendly title to put at the top of the page.
	Summary() string                     // Human-friendly summary to put at the top of the page (maybe)
	Permalink() string                   // Permanent link to the record being built
	BasePath() string                    // URL Path of the root of this object, without any additional actions.
	URL() string                         // Complete URL of the requested page
	QueryParam(string) string            // Query parameter of the requested page
	SetQueryParam(string, string) bool   // Sets a queryString parameter
	IsAuthenticated() bool               // Returns TRUE if the user is signed in
	IsOwner() bool                       // Returns TRUE if the signed-in user is the owner of this object
	IsAdminBuilder() bool                // Returns TRUE if this is an admin route
	IsPartialRequest() bool              // Returns TRUE if this is an HTMX request for a page fragment
	UserCan(string) bool                 // Returns TRUE if the signed-in user has access to the named action
	AuthenticatedID() primitive.ObjectID // Returns the ID of the signed-in user (or zero if not signed in)
	Search() SearchBuilder               // Returns a SearchBuilder for this Builder

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

	factory() Factory                      // The service factory
	request() *http.Request                // The original http.Request that we are responding to
	response() http.ResponseWriter         // The original http.ResponseWriter that we are responding to
	authorization() model.Authorization    // The user's authorization data from the context
	service() service.ModelService         // The abstracted ModelService the backs this Builder
	templateRole() string                  // Returns the role that the current template plays in the system. Used for choosing child template.
	objectType() string                    // The type of object being built
	schema() schema.Schema                 // Schema to use to validate this Object
	object() data.Object                   // Model Object being built
	objectID() primitive.ObjectID          // MongoDB ObjectID of the Object being built
	getUser() (*model.User, error)         // Retrieves the currently-logged-in user
	getIdentity() (*model.Identity, error) // Retrieves the currently-logged-in user
	lookupProvider() form.LookupProvider   // Retrieves the LookupProvider for this user

	actionID() string                     // Token that identifies the action requested via the URL.
	actions() map[string]model.Action     // Map of actions that are available for this object
	action() model.Action                 // The pipeline action to be taken by this builder
	execute(io.Writer, string, any) error // The HTML template used by this Builder

	clone(action string) (Builder, error) // Creates a new Builder with the same type and object, but a different action
	debug()                               // Outputs debug information to the console
}

type templateGetter interface {
	template() model.Template
}
