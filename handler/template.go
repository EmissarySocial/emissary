package handler

import (
	"bytes"
	"net/http"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// executeThemeTemplate renders a named template from this Domain's Theme and writes it to
// the response.  It is the single entry point for pages served directly by a handler
// instead of by a Template action.
func executeThemeTemplate(ctx *steranko.Context, factory *service.Factory, templateName string, page any) error {

	const location = "handler.executeThemeTemplate"

	// RULE: `page` must be a build.Theme, or a Builder that embeds one -- never a map or a
	// bare model.  The Theme's shared partials resolve their accessors at RENDER time, so a
	// wrong dot 500s the page instead of failing the build.  TestThemeTemplateDots is what
	// enforces this: it maps every handler-rendered template to its dot and fails if one is
	// unmapped.
	var buffer bytes.Buffer

	template := factory.Domain().Theme().HTMLTemplate

	// Render into a buffer, so a mid-render failure returns an error instead of
	// committing a half-written page to the response.
	if err := template.ExecuteTemplate(&buffer, templateName, page); err != nil {
		return derp.Wrap(err, location, "Executing template", templateName)
	}

	// Write the finished page to the response
	return ctx.HTML(http.StatusOK, buffer.String())
}

// executeDomainTemplate renders a named template from this Domain's Theme, using a plain
// Theme builder as its dot.
func executeDomainTemplate(ctx *steranko.Context, factory *service.Factory, session data.Session, templateName string) error {

	builder := build.NewTheme(factory, session, ctx.Request(), ctx.Response())

	return executeThemeTemplate(ctx, factory, templateName, builder)
}
