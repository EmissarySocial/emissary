package builder

import (
	"io"

	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
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

	spew.Dump(providerID, adapter, connection, err)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error loading connection", providerID))
	}

	// Try to find a Manual Provider for this Provider
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return Halt().WithError(derp.NewInternalError(location, "Provider does not implement ManualProvider interface", adapter))
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

	postData := mapof.NewAny()

	if err := bind(builder.request(), &postData); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing POST data"))
	}

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
		return Halt().WithError(derp.NewInternalError(location, "Provider does not implement ManualProvider interface", adapter))
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Apply the form data to the domain object
	if err := form.SetAll(&connection, postData, nil); err != nil {
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
