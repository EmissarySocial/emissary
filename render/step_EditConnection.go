package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

type StepEditConnection struct{}

func (step StepEditConnection) Get(renderer Renderer, buffer io.Writer) error {

	const location = "render.StepEditConnection.Get"

	// This step must be run in a Domain admin
	domainRenderer := renderer.(*Domain)

	// Collect parameters and services
	context := renderer.context()
	providerID := context.Request().URL.Query().Get("provider")

	client := domainRenderer.Client(providerID)
	adapter := domainRenderer.Provider(providerID)

	// Try to find a Manual Provider for this Provider
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return derp.NewInternalError(location, "Provider does not implement ManualProvider interface", adapter)
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Write the form data
	result, err := form.Editor(client, nil)

	if err != nil {
		return derp.Wrap(err, location, "Error generating form editor")
	}

	// Wrap the form as a ModalForm and return
	result = WrapModalForm(context.Response(), renderer.URL(), result)
	buffer.Write([]byte(result))

	return nil
}

func (step StepEditConnection) Post(renderer Renderer, _ io.Writer) error {

	const location = "render.StepEditConnection.Post"

	// This step must be run in a Domain admin
	domainRenderer := renderer.(Domain)
	context := renderer.context()

	postData := mapof.NewAny()

	if err := context.Bind(&postData); err != nil {
		return derp.Wrap(err, location, "Error parsing POST data")
	}

	// Collect parameters and services
	providerID := context.Request().URL.Query().Get("provider")

	client := domainRenderer.Client(providerID)
	adapter := domainRenderer.Provider(providerID)

	// Try to find a Manual Provider for this Provider
	manualProvider, ok := adapter.(providers.ManualProvider)

	if !ok {
		return derp.NewInternalError(location, "Provider does not implement ManualProvider interface", adapter)
	}

	// Retrieve the custom form for this Manual Provider
	form := manualProvider.ManualConfig()

	// Apply the form data to the domain object
	if err := form.SetAll(&client, postData, nil); err != nil {
		return derp.Wrap(err, location, "Error updating domain object form")
	}

	// Run post-configuration scripts, if any
	if err := adapter.AfterConnect(renderer.factory(), &client); err != nil {
		return derp.Wrap(err, location, "Error installing client")
	}

	// Prevent nil maps
	if domainRenderer.domain.Clients == nil {
		domainRenderer.domain.Clients = make(set.Map[model.Client])
	}

	domainRenderer.domain.Clients.Put(client)

	// Try to save the domain object back to the database
	domainService := domainRenderer.domainService()

	if err := domainService.Save(domainRenderer.domain, "Updated connection"); err != nil {
		return derp.Wrap(err, location, "Error saving domain object")
	}

	CloseModal(context, "")
	return nil
}

func (step StepEditConnection) UseGlobalWrapper() bool {
	return false
}
