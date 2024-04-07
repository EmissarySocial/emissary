package builder

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

type StepEditConnection struct{}

func (step StepEditConnection) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditConnection.Get"

	// This step must be run in a Domain admin
	domainBuilder := builder.(*Domain)

	// Collect parameters and services
	providerID := builder.QueryParam("provider")

	client := domainBuilder.Client(providerID)
	adapter := domainBuilder.Provider(providerID)

	// Try to find a Manual Provider for this Provider
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "Provider does not implement ManualProvider interface", adapter))
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Write the form data
	formHTML, err := form.Editor(client, nil)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error generating form editor"))
	}

	// Wrap the form as a ModalForm and return
	formHTML = WrapModalForm(builder.response(), builder.URL(), form.Encoding(), formHTML)

	// nolint:errcheck
	buffer.Write([]byte(formHTML))

	return Halt().AsFullPage()
}

func (step StepEditConnection) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditConnection.Post"

	// This step must be run in a Domain admin
	domainBuilder := builder.(Domain)

	postData := mapof.NewAny()

	if err := bind(builder.request(), &postData); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing POST data"))
	}

	// Collect parameters and services
	providerID := builder.QueryParam("provider")

	client := domainBuilder.Client(providerID)
	adapter := domainBuilder.Provider(providerID)

	// Try to find a Manual Provider for this Provider
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "Provider does not implement ManualProvider interface", adapter))
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Apply the form data to the domain object
	if err := form.SetAll(&client, postData, nil); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error updating domain object form"))
	}

	// Run post-configuration scripts, if any
	if err := adapter.AfterConnect(builder.factory(), &client); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error installing client"))
	}

	// Prevent nil maps
	if domainBuilder.domain.Clients == nil {
		domainBuilder.domain.Clients = make(set.Map[model.Client])
	}

	domainBuilder.domain.Clients.Put(client)

	// Try to save the domain object back to the database
	domainService := domainBuilder.domainService()

	if err := domainService.Save(*domainBuilder._domain, "Updated connection"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving domain object"))
	}

	return Halt().WithEvent("closeModal", "").AsFullPage()
}
