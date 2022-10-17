package render

import (
	"io"

	"github.com/EmissarySocial/emissary/service/external"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
	"github.com/davecgh/go-spew/spew"
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
	adapter := domainRenderer.Adapter(providerID)

	// Try to find a Manual Adapter for this Provider
	manualAdapter, ok := adapter.(external.ManualAdapter)

	if !ok {
		return derp.NewInternalError(location, "Adapter does not implement ManualAdapter interface", adapter)
	}

	// Retrieve the custom form for this Manual Adapter
	form := manualAdapter.ManualConfig()

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

func (step StepEditConnection) Post(renderer Renderer) error {

	const location = "render.StepEditConnection.Post"

	// This step must be run in a Domain admin
	domainRenderer := renderer.(Domain)
	context := renderer.context()

	postData := maps.New()

	if err := context.Bind(&postData); err != nil {
		return derp.Wrap(err, location, "Error parsing POST data")
	}

	// Collect parameters and services
	providerID := context.Request().URL.Query().Get("provider")

	client := domainRenderer.Client(providerID)
	adapter := domainRenderer.Adapter(providerID)

	// Try to find a Manual Adapter for this Provider
	manualAdapter, ok := adapter.(external.ManualAdapter)

	if !ok {
		return derp.NewInternalError(location, "Adapter does not implement ManualAdapter interface", adapter)
	}

	// Retrieve the custom form for this Manual Adapter
	form := manualAdapter.ManualConfig()

	spew.Dump(postData, client)

	// Apply the form data to the domain object
	if err := form.Do(postData, &client); err != nil {
		return derp.Wrap(err, location, "Error updating domain object form")
	}

	domainRenderer.domain.Clients.Put(client)

	// Try to save the domain object back to the database
	domainService := domainRenderer.domainService()

	if err := domainService.Save(domainRenderer.domain, "Updated connection"); err != nil {
		return derp.Wrap(err, location, "Error saving domain object")
	}

	// TODO: Call the "INSTALL" feature of the adapter

	CloseModal(context, "")
	return nil
}

func (step StepEditConnection) UseGlobalWrapper() bool {
	return false
}
