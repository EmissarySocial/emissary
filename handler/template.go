package handler

import (
	"bytes"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

func executeDomainTemplate(ctx *steranko.Context, factory *domain.Factory, templateName string) error {

	const location = "handler.executeDomainTemplate"

	var buffer bytes.Buffer

	domain := factory.Domain().Get()

	// Find and execute the template
	template := factory.Domain().Theme().HTMLTemplate

	if err := template.ExecuteTemplate(&buffer, templateName, &domain); err != nil {
		return derp.Wrap(err, location, "Error executing template")
	}

	// Write the result to the response.
	return ctx.HTML(200, buffer.String())
}
