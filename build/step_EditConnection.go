package build

import (
	"io"

	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/benpate/derp"
)

type StepEditConnection struct{}

func (step StepEditConnection) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditConnection.Get"

	// This step must be run in a Domain admin
	domainBuilder := builder.(Domain)

	// Collect parameters and services
	factory := domainBuilder.factory()
	connectionService := factory.Connection()
	providerID := builder.QueryParam("providerId")
	adapter := domainBuilder.Provider(providerID)

	connection, err := connectionService.LoadOrCreateByProvider(providerID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error loading connection", providerID))
	}

	// Try to find a Manual Provider for this Provider
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return Halt().WithError(derp.InternalError(location, "Provider does not implement ManualProvider interface", adapter))
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Write the form data
	formHTML, err := form.Editor(&connection, nil)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error generating form editor"))
	}

	// Wrap the form as a ModalForm and return
	formHTML = WrapModalForm(builder.response(), builder.URL(), formHTML, form.Encoding())

	// nolint:errcheck
	buffer.Write([]byte(formHTML))

	return Halt().AsFullPage()
}

func (step StepEditConnection) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditConnection.Post"

	// This step must be run in a Domain admin
	domainBuilder := builder.(Domain)

	// Collect parameters and services
	providerID := builder.QueryParam("providerId")

	factory := domainBuilder.factory()
	connectionService := factory.Connection()
	adapter := domainBuilder.Provider(providerID)

	connection, err := connectionService.LoadOrCreateByProvider(providerID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error loading connection", providerID))
	}

	// To manually configure a connection, it must be a "ManualProvider".  Other types,
	// like OAuth Providers are handled separately
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return Halt().WithError(derp.InternalError(location, "Provider does not implement ManualProvider interface", adapter))
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Parse the data in the Form post
	if err := builder.request().ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form body"))
	}

	// Apply the form data to the domain object
	if err := form.SetURLValues(&connection, builder.request().Form, nil); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error updating domain object form"))
	}

	// Run post-configuration scripts, if any
	if err := adapter.AfterConnect(builder.factory(), &connection); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error installing connection"))
	}

	// Try to save the domain object back to the database
	if err := connectionService.Save(&connection, "Updated by Administrator"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error saving domain object"))
	}

	return Halt().WithEvent("closeModal", "").WithEvent("refreshPage", "").AsFullPage()
}
