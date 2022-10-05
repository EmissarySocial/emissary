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
		return derp.Report(derp.Wrap(err, location, "Error getting factory"))
	}

	// Find and execute the template
	template := factory.Layout().Global().HTMLTemplate

	if err := template.ExecuteTemplate(&buffer, templateName, &domain); err != nil {
		return derp.Report(derp.Wrap(err, location, "Error executing template"))
	}

	// Write the result to the response.
	return ctx.HTML(200, buffer.String())
}

func loadFactoryAndDomain(fm *server.Factory, ctx echo.Context) (*domain.Factory, model.Domain, error) {

	const location = "handler.loadFactoryAndDomain"

	// Try to locate the factory for this domain
	factory, err := fm.ByContext(ctx)

	if err != nil {
		return nil, model.Domain{}, derp.Report(derp.Wrap(err, location, "Error getting factory"))
	}

	// Try to load the domain record
	domainService := factory.Domain()
	domain := model.NewDomain()

	if err := domainService.Load(&domain); err != nil {
		return nil, model.Domain{}, derp.Report(derp.Wrap(err, location, "Error loading domain record"))
	}

	return factory, domain, nil

}
