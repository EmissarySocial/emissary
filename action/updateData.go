package action

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
	"github.com/benpate/steranko"
)

// UpdateData updates the specific data in a stream
type UpdateData struct {
	streamService *service.Stream
	Form          form.Form
	Info
}

// Get displays a form where users can update stream data
func (action UpdateData) Get(ctx steranko.Context, factory *domain.Factory, stream *model.Stream) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (action UpdateData) Post(ctx steranko.Context, factory *domain.Factory, stream *model.Stream) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.action.UpdateData.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := action.Form.AllPaths()
	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := stream.SetPath(p, body); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.action.UpdateData.Post", "Error seting value", field)
		}
	}

	// Try to update the stream
	streamService := factory.Stream()
	if err := streamService.Save(stream, "Moved to new State"); err != nil {
		return derp.Wrap(err, "ghost.action.MoveState.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	ctx.Request().Header.Add("HX-Redirect", "/"+stream.Token)
	return ctx.NoContent(http.StatusOK)
}
