package render

import (
	"html/template"
	"io"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/schema"
	"github.com/benpate/steranko"
)

// Renderer safely wraps model objects for consumption by an html Template
type Renderer interface {
	Render() (template.HTML, error) // Render function outputs an HTML template
	Token() string                  // URL Token of the record being rendered
	URL() string
	ActionID() string             // The ID of the action to be taken by this renderer
	Action() (model.Action, bool) // The pipeline action to be taken by this renderer
	IsPartialRequest() bool       // Returns TRUE if this is an HTMX request for a page fragment
	factory() Factory             // The service factory
	context() *steranko.Context   // The request context embedded in the Renderer
	object() data.Object          // Model Object being rendered
	schema() schema.Schema        // Schema to use to validate this Object
	service() ModelService

	executeTemplate(io.Writer, string, interface{}) error // The HTML template used by this Renderer

	path.Setter
	path.Getter
}

func NewRenderer(factory Factory, ctx *steranko.Context, object data.Object, actionID string) (Renderer, error) {

	switch obj := object.(type) {

	case *model.Group:
		result := NewGroup(factory, ctx, obj, actionID)
		return &result, nil

	case *model.Stream:
		result, err := NewStreamWithoutTemplate(factory, ctx, obj, actionID)
		return &result, err

	case *model.User:
		result := NewUser(factory, ctx, obj, actionID)
		return &result, nil
	}

	return nil, derp.New(derp.CodeInternalError, "ghost.render.NewRenderer", "Unrecognized object", object)
}
