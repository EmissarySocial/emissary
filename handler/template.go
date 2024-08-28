package handler

import (
	"bytes"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
)

func executeDomainTemplate(fm *server.Factory, ctx echo.Context, templateName string) error {

	const location = "handler.executeDomainTemplate"

	var buffer bytes.Buffer

	// Try to load the factory and domain
	factory, domain, err := loadFactoryAndDomain(fm, ctx)

	if err != nil {
		return derp.Wrap(err, location, "Error getting factory")
	}

	// Find and execute the template
	template := factory.Domain().Theme().HTMLTemplate

	if err := template.ExecuteTemplate(&buffer, templateName, &domain); err != nil {
		return derp.Wrap(err, location, "Error executing template")
	}

	// Write the result to the response.
	return ctx.HTML(200, buffer.String())
}

// TODO: This should be refactored away using With* wrappers.
func loadFactoryAndDomain(fm *server.Factory, ctx echo.Context) (*domain.Factory, model.Domain, error) {

	const location = "handler.loadFactoryAndDomain"

	// Try to locate the factory for this domain
	factory, err := fm.ByContext(ctx)

	if err != nil {
		return nil, model.Domain{}, derp.Wrap(err, location, "Error getting factory")
	}

	// Get the domain record
	domain := factory.Domain().Get()

	return factory, domain, nil
}
