package handler

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/service"
)

func renderPage(layoutService *service.Layout, renderer render.Renderer, fullPage bool) (string, error) {

	template := layoutService.Layout()

	body, err := renderer.Render()

	if err != nil {
		return "", derp.Wrap(err, "ghost.handler.fullPage", "Error rendering page body")
	}

	if fullPage {

		var result strings.Builder

		// Render Header
		if err := template.ExecuteTemplate(&result, "page-header", renderer); err != nil {
			return "", derp.Wrap(err, "ghost.handler.fullPage", "Error rendering page header")
		}

		if _, err := result.WriteString(body); err != nil {
			return "", derp.Wrap(err, "ghost.handler.fullPage", "Error writing body to buffer")
		}

		// Render Footer
		if err := template.ExecuteTemplate(&result, "page-footer", renderer); err != nil {
			return "", derp.Wrap(err, "ghost.handler.fullPage", "Error rendering page header")
		}

		return result.String(), nil
	}

	return body, nil
}
