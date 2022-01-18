package render

import (
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
	"github.com/whisperverse/whisperverse/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Renderer safely wraps model objects for consumption by an html Template
type Renderer interface {
	Render() (template.HTML, error) // Render function outputs an HTML template

	path.Setter
	path.Getter

	// COMMON API METHODS
	Token() string                // URL Token of the record being rendered
	URL() string                  // Complete URL of the requested page
	ActionID() string             // The ID of the action to be taken by this renderer
	Action() *model.Action        // The pipeline action to be taken by this renderer
	IsPartialRequest() bool       // Returns TRUE if this is an HTMX request for a page fragment
	factory() Factory             // The service factory
	context() *steranko.Context   // The request context embedded in the Renderer
	service() ModelService        // The abstracted ModelService the backs this Renderer
	schema() schema.Schema        // Schema to use to validate this Object
	object() data.Object          // Model Object being rendered
	objectID() primitive.ObjectID // MongoDB ObjectID of the Object being rendered

	executeTemplate(io.Writer, string, interface{}) error // The HTML template used by this Renderer
}

func NewRenderer(factory Factory, ctx *steranko.Context, object data.Object, actionID string) (Renderer, error) {

	switch obj := object.(type) {

	case *model.Group:
		layout := factory.Layout().Group()
		action := layout.Action(actionID)
		result := NewGroup(factory, ctx, layout, action, obj)
		return &result, nil

	case *model.Stream:
		result, err := NewStreamWithoutTemplate(factory, ctx, obj, actionID)
		return &result, err

	case *model.User:
		layout := factory.Layout().User()
		action := layout.Action(actionID)
		result := NewUser(factory, ctx, layout, action, obj)
		return &result, nil
	}

	return nil, derp.New(derp.CodeInternalError, "whisper.render.NewRenderer", "Unrecognized object", object)
}

/*******************************************
 * ADDITIONAL DATA
 *******************************************/

// AdminSections returns labels and values for all hard-coded sections of the administrator area.
func AdminSections() []model.Option {
	return []model.Option{
		{
			Value: "domain",
			Label: "Site",
		},
		{
			Value: "toplevel",
			Label: "Navigation",
		},
		{
			Value: "users",
			Label: "People",
		},
		{
			Value: "groups",
			Label: "Groups",
		},
		{
			Value: "analytics",
			Label: "Analytics",
		},
	}
}
