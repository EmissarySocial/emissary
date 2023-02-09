package render

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepIfActivityPub represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepIfActivityPub struct {
	View    string
	Dataset string
}

// Get displays a form where users can update stream data
func (step StepIfActivityPub) Get(renderer Renderer, buffer io.Writer) error {

	context := renderer.context()

	// If this is not an ActivityPub request, then do nothing.
	if !step.isActivityPub(context.Request()) {
		return nil
	}

	object := renderer.object()

	// Otherwise, render JSON-LD data defined by step args
	switch step.View {

	case "object":

		if getter, ok := object.(model.JSONLDGetter); ok {
			context.Response().Header().Set("Content-Type", "application/activity+json")
			context.JSON(http.StatusOK, getter.GetJSONLD())
			return HaltPipeline()
		}

		return derp.NewInternalError("render.StepIfActivityPub.Get", "Object must implement JSONLDGetter interface", step, object)

	case "collection":
		return derp.NewInternalError("render.StepIfActivityPub.Get", "ActivityPub Collection not yet implemented", nil, step, object)
	}

	return derp.NewBadRequestError("render.StepIfActivityPub.Get", "Error generating ActivityPub data", step, object)
}

// HaltPipeline optionally allows this action to stop processing the action pipeline.
func (step StepIfActivityPub) HaltPipeline(renderer Renderer) bool {

	// If this is an ActivityPub request, then you should totally halt the pipeline
	return step.isActivityPub(renderer.context().Request())
}

func (step StepIfActivityPub) UseGlobalWrapper() bool {
	return false
}

// Post updates the stream with approved data from the request body.
func (step StepIfActivityPub) Post(renderer Renderer) error {
	return derp.NewNotFoundError("render.StepIfActivityPub.Post", "Cannot POST to this resource", nil)
}

// isActivityPub returns TRUE if the request is an ActivityPub request
func (step StepIfActivityPub) isActivityPub(request *http.Request) bool {
	return request.Header.Get("Accept") == "application/activity+json"
}
