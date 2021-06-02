package action

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
	"github.com/benpate/steranko"
)

// UpdateData updates the specific data in a stream
type UpdateData struct {
	config          model.ActionConfig
	templateService *service.Template
	streamService   *service.Stream
	formLibrary     form.Library
}

func NewAction_UpdateData(config model.ActionConfig, templateService *service.Template, streamService *service.Stream, formLibrary form.Library) UpdateData {

	return UpdateData{
		config: config,

		templateService: templateService,
		streamService:   streamService,
		formLibrary:     formLibrary,
	}
}

// Get displays a form where users can update stream data
func (action *UpdateData) Get(ctx steranko.Context, stream *model.Stream) error {

	schema, err := action.templateService.Schema(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.service.Stream.Form", "Invalid Schema")
	}

	result, err := action.Form.HTML(action.formLibrary, schema, stream)

	if err != nil {
		return derp.Wrap(err, "ghost.service.Stream.Form", "Error generating form")
	}

	return ctx.HTML(http.StatusOK, result)
}

// Post updates the stream with approved data from the request body.
func (action *UpdateData) Post(ctx steranko.Context, stream *model.Stream) error {

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
	if err := action.streamService.Save(stream, "Moved to new State"); err != nil {
		return derp.Wrap(err, "ghost.action.MoveState.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+stream.Token+`"}}`)
	return ctx.NoContent(http.StatusOK)
}

// Config returns the configuration information for this action
func (action *UpdateData) Config() model.ActionConfig {
	return action.config
}
