package build

import (
	"io"

	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepEditConnection is a Step that edits a Domain's connection to a third-party service
type StepEditConnection struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepEditConnection) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditConnection.Get"

	// This step must be run in a Domain admin
	domainBuilder := builder.(Domain)

	// Collect parameters and services
	factory := domainBuilder.factory()
	connectionService := factory.Connection()
	providerID := builder.QueryParam("providerId")
	adapter := domainBuilder.Provider(providerID)

	connection, err := connectionService.LoadOrCreateByProvider(builder.session(), providerID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading connection", providerID))
	}

	// Retrieve the custom form for this Manual Provider
	settingsForm, err := providerSettingsForm(adapter)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting settings form", providerID))
	}

	// Write the form data
	formHTML, err := settingsForm.Editor(
		&connection,
		factory.LookupProvider(
			builder.request(),
			builder.session(),
			builder.AuthenticatedID(),
		),
	)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Generating form editor"))
	}

	// Wrap the form as a ModalForm and return
	formHTML = WrapModalForm(builder.response(), builder.RelativeURL(), formHTML, settingsForm.Encoding())

	if _, err := buffer.Write([]byte(formHTML)); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Writing form HTML to buffer"))
	}

	return Halt().AsFullPage()
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepEditConnection) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditConnection.Post"

	// This step must be run in a Domain admin
	domainBuilder := builder.(Domain)

	// Collect parameters and services
	providerID := builder.QueryParam("providerId")

	factory := domainBuilder.factory()
	connectionService := factory.Connection()
	adapter := domainBuilder.Provider(providerID)

	connection, err := connectionService.LoadOrCreateByProvider(builder.session(), providerID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading connection", providerID))
	}

	// Retrieve the custom form for this Manual Provider
	settingsForm, err := providerSettingsForm(adapter)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting settings form", providerID))
	}

	// Parse the data in the Form post
	if err := builder.request().ParseForm(); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Parsing form body"))
	}

	// Apply the form data to the domain object
	if err := settingsForm.SetURLValues(&connection, builder.request().Form, nil); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Updating domain object with form data"))
	}

	// Try to save the domain object back to the database
	if err := connectionService.Save(builder.session(), &connection, "Updated by Administrator"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving domain object"))
	}

	return Halt().WithEvent("closeModal", "").WithEvent("refreshPage", "").AsFullPage()
}

// providerSettingsForm returns the settings form for a Provider, refusing one whose elements and
// schema disagree. A form that writes a path its schema omits drops that field on save, silently.
func providerSettingsForm(adapter providers.Provider) (form.Form, error) {

	const location = "build.providerSettingsForm"

	// To manually configure a connection, it must be a "ManualProvider".  Other types,
	// like OAuth Providers are handled separately
	manualProvider, isManualProvider := adapter.(providers.ManualProvider)

	if !isManualProvider {
		return form.Form{}, derp.Internal(location, "Provider does not implement ManualProvider interface", adapter)
	}

	result := manualProvider.ManualConfig()

	if err := result.Validate(); err != nil {
		return form.Form{}, derp.Wrap(err, location, "Provider settings form does not match its own schema")
	}

	return result, nil
}
