package render

import (
	"html/template"

	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Renderer wraps the Render method, which returns an HTML rendering of an object.
type Renderer interface {
	// Render returns an HTML rendering of an object.
	Render() (string, error)
}

type LayoutService interface {
	Layout() *template.Template
}

type TemplateService interface {
	Load(string) (*model.Template, error)
	LoadCompiled(string, string, string) (*model.Template, *template.Template, error)
	ListByContainer(string) []model.Template
}

type StreamService interface {
	New() *model.Stream
	LoadByToken(string) (*model.Stream, error)
	LoadParent(*model.Stream) (*model.Stream, error)
	List(expression.Expression, ...option.Option) (data.Iterator, error)
	ListByParent(primitive.ObjectID) (data.Iterator, error)
	ListByFolder(primitive.ObjectID) (data.Iterator, error)
}
