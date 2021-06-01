package action

import (
	"bytes"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/steranko"
)

type ViewStream struct {
	CommonInfo

	layoutService *service.Layout
}

func NewAction_ViewStream(config *model.ActionConfig, layoutService *service.Layout) ViewStream {
	return ViewStream{
		CommonInfo:    NewCommonInfo(config),
		layoutService: layoutService,
	}
}

// Get renders the Stream HTML to the context
func (action ViewStream) Get(ctx steranko.Context, stream *model.Stream) error {

	var result bytes.Buffer

	renderer := factory.StreamViewer(ctx, *stream, action.CommonInfo.ActionID)

	// Partial page requests
	if renderer.Partial() {

		if html, err := renderer.Render(); err == nil {
			return ctx.HTML(200, string(html))
		} else {
			return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering partial HTML template")
		}
	}

	// Render full page
	template := action.layoutService.Template

	if err := template.ExecuteTemplate(&result, "page", renderer); err != nil {
		return derp.Wrap(err, "ghost.handler.renderStream", "Error rendering HTML template")
	}

	return ctx.HTML(200, result.String())
}

// Post is not supported for this action.
func (action ViewStream) Post(ctx steranko.Context, stream *model.Stream) error {
	return derp.New(derp.CodeBadRequestError, "ghost.action.ViewStream.Post", "Unsupported Method")
}
