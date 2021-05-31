package action

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

type ViewStream struct {
	streamService *service.Stream
	Info
}

func (action ViewStream) Get(ctx echo.Context, stream *model.Stream) (string, error) {

	var result bytes.Buffer

	renderer := domain.NewRenderer(action.streamService, request, *stream)
	renderer.view = action.Info.ActionID

	// Partial page requests (stream only)
	if renderer.Partial() {

		if html, err := renderer.Render(); err == nil {
			return ctx.HTML(200, string(html))
		} else {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering partial HTML template")
		}
	}

	// Render full page (stream only).
	layoutService := factory.Layout()
	template := layoutService.Template

	if err := template.ExecuteTemplate(&result, "page", renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
	}

	return ctx.HTML(200, result.String())

}

func (action ViewStream) Post(request domain.HTTPRequest, stream *model.Stream) (string, error) {
	return "", nil
}
