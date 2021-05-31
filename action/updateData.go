package action

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/middleware"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
)

// UpdateData updates the specific data in a stream
type UpdateData struct {
	streamService *service.Stream
	form          form.Form
	Info
}

func (action UpdateData) Get(ctx middleware.GhostContext, stream *model.Stream) error {
	return nil
}

func (action UpdateData) Post(ctx middleware.GhostContext, stream *model.Stream) error {

	body := datatype.Map{}

	allPaths := action.form.AllPaths()

	if err := ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.action.UpdateData.Post", "Error binding body")
	}

	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := stream.SetPath(p, body); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.action.UpdateData.Post", "Error seting value", field)
		}
	}

	return nil
}
