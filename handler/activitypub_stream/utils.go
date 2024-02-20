package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

// getActor wraps all of the monotonous code of loading a factory, templateService, streamService, along with the Template, Stream, and Actor
// from the Request context.
func getActor(serverFactory *server.Factory, ctx echo.Context) (*domain.Factory, *service.Template, *service.Stream, model.Template, model.Stream, model.Actor, error) {

	const location = "activitypub_stream.getActor"

	factory, err := serverFactory.ByContext(ctx)

	if err != nil {
		return nil, nil, nil, model.Template{}, model.Stream{}, model.Actor{}, derp.Wrap(err, location, "Unrecognized Domain")
	}

	// Try to load the Stream
	streamService := factory.Stream()
	stream := model.NewStream()
	token := ctx.Param("stream")
	if err := streamService.LoadByToken(token, &stream); err != nil {
		return nil, nil, nil, model.Template{}, model.Stream{}, model.Actor{}, derp.Wrap(err, location, "Error loading stream", token)
	}

	// Try to load the Stream's Template
	templateService := factory.Template()
	template, err := templateService.Load(stream.TemplateID)
	if err != nil {
		return nil, nil, nil, model.Template{}, model.Stream{}, model.Actor{}, derp.Wrap(err, location, "Invalid Template", stream.TemplateID)
	}

	// Validate the Actor for this request
	actor := template.Actor
	if actor.IsNil() {
		return nil, nil, nil, model.Template{}, model.Stream{}, model.Actor{}, derp.NewNotFoundError(location, "Actor not found", stream)
	}

	return factory, templateService, streamService, template, stream, actor, nil
}
