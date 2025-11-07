package build

import (
	"io"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/mapof"
)

// StepEditRegistration is a Step that can update the data.DataMap custom data stored in a Stream
type StepEditRegistration struct{}

func (step StepEditRegistration) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "builder.StepEditRegistration.Get"

	// Require that this is only run in a Domain Builder
	domainBuilder, ok := builder.(Domain)

	if !ok {
		return Halt().WithError(derp.InternalError(location, "Step edit-registration can only be used in an admin/domain template"))
	}

	factory := builder.factory()
	registrationService := factory.Registration()
	options := registrationService.List()
	registrationID := builder.QueryParam("registrationId")

	b := html.New()

	b.H1().ID("modal-title").InnerText("User Registration Options").Close()
	b.Form("", "").Attr("hx-get", "/admin/domain/signup").Attr("hx-trigger", "change").Attr("hx-push-url", "false")
	b.Div().Class("layout layout-vertical")
	b.Div().Class("layout-elements")
	b.Div().Class("layout-element")
	b.Label("select-template").InnerText("Registration Method...").Close()
	b.Select("registrationId").ID("select-template").TabIndex("0")

	if registrationID == "" {
		b.OptionSelected("Not Allowed", "")
	} else {
		b.Option("Not Allowed", "")
	}

	for _, option := range options {
		if option.Value == registrationID {
			b.OptionSelected(option.Label, option.Value)
		} else {
			b.Option(option.Label, option.Value)
		}
	}

	b.CloseAll()
	result := b.String()

	registration, err := registrationService.Load(registrationID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load registration template", registrationID))
	}

	if registration.IsZero() {
		b.Button().Type("button").Class("primary").Attr("hx-post", "/admin/domain/signup?registrationId=").Attr("hx-swap", "none").InnerText("Save Changes").Close()
		b.Button().Script("on click send closeModal").InnerText("Cancel").Close()
		result = b.String()

	} else {
		userID := builder.authorization().UserID
		lookupProvider := factory.LookupProvider(builder.request(), builder.session(), userID)

		f := form.New(registration.Schema, registration.Form)
		formHTML, err := f.Editor(domainBuilder._domain.RegistrationData, lookupProvider)

		if err != nil {
			return Halt().WithError(derp.Wrap(err, "builder.StepEditRegistration", "Error building registration form"))
		}

		result += WrapForm("/admin/domain/signup?registrationId="+registrationID, formHTML, "")
	}

	result = WrapModal(builder.response(), result)

	if _, err := buffer.Write([]byte(result)); err != nil {
		return Halt().WithError(derp.Wrap(err, "builder.StepEditRegistration", "Error writing response buffer"))
	}

	return Halt()
}

// Post updates the stream with approved data from the request body.
func (step StepEditRegistration) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "builder.StepEditRegistration.Post"

	// Require that this is only run in a Domain Builder
	domainBuilder, ok := builder.(Domain)

	if !ok {
		return Halt().WithError(derp.InternalError(location, "Step edit-registration can only be used in an admin/domain template"))
	}

	factory := builder.factory()

	// Collect variables for this transaction
	registrationID := domainBuilder.QueryParam("registrationId")

	// If the registrationID is empty, then we are disabling signups
	if registrationID == "" {
		domainBuilder._domain.RegistrationID = ""
		domainBuilder._domain.RegistrationData = mapof.NewString()
		return Continue().WithEvent("closeModal", "true")
	}

	// Otherwise, load and validate the Template
	registration, err := factory.Registration().Load(registrationID)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Unable to load template", registrationID))
	}

	// Bind to the form POST data
	inputs, err := formdata.Parse(builder.request())
	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error binding form data"))
	}

	// Use the schema to set the form inputs into a new map
	data := mapof.NewString()
	if err := registration.Schema.SetURLValues(&data, inputs); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error updating domain object form"))
	}

	if data.GetString("secret") == "" {
		secret, _ := random.GenerateString(32)
		data["secret"] = secret
	}

	// Apply the new values to the domain object
	domainBuilder._domain.RegistrationID = registrationID
	domainBuilder._domain.RegistrationData = data

	// Success. (close the modal)
	return Continue().WithEvent("closeModal", "true")
}
