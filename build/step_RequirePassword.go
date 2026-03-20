package build

import (
	"io"
	"net/http"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepRequirePassword is a Step that requires the user to enter their password before performing a sensitive action
type StepRequirePassword struct {
	Title       *template.Template
	Message     *template.Template
	Submit      string
	SubmitClass string
	Cancel      string
}

func (step StepRequirePassword) Get(builder Builder, _ io.Writer) PipelineBehavior {

	b := html.New()

	b.H1().InnerText(executeTemplate(step.Title, builder)).Close()
	b.Div().Class("margin-bottom").InnerHTML(executeTemplate(step.Message, builder)).Close()

	b.Form("", "").
		Attr("hx-post", builder.URL()).
		Attr("hx-swap", "none").
		Attr("hx-push-url", "false").
		Script("on submit set #htmx-response-message.innerHTML to ''")

	b.Div().Class("layout-vertical")
	b.Div().Class("layout-elements")
	b.Div().Class("layout-element")
	b.Label("idConfirmPassword").InnerText("Confirm Your Password to Continue").Close()
	b.Input("password", "confirm_password").ID("idConfirmPassword").Close()
	b.Close()
	b.Close()
	b.Close()

	b.Button().Type("submit").Class(step.SubmitClass, "htmx-request-hide").
		InnerText(step.Submit).
		Close()

	b.Button().Class(step.SubmitClass, "htmx-request-show").Attr("disabled", "true")
	b.Span().Class("spin")
	b.I("bi", "bi-arrow-clockwise").Close()
	b.Close()
	b.Span().Class("margin-left-sm").InnerText(step.Submit).Close()
	b.Close()

	b.Button().Script("on click trigger closeModal").InnerText(step.Cancel).Close()
	b.Span().ID("htmx-response-message").Class("margin-left").Close()
	b.CloseAll()

	modalHTML := WrapModal(builder.response(), b.String())

	if _, err := io.WriteString(builder.response(), modalHTML); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepRequirePassword.Get", "Unable to write modal HTML to response"))
	}

	return Halt().AsFullPage()

}

// Post updates the stream with the current date as the "PublishDate"
func (step StepRequirePassword) Post(builder Builder, writer io.Writer) PipelineBehavior {

	// RULE: Require authentication to publish content
	if !builder.IsAuthenticated() {
		return step.error(builder, "Invalid user. Sign out and try again.")
	}

	// Load the currently signed in user from the database
	factory := builder.factory()

	userService := factory.User()
	user := model.NewUser()
	authorization := builder.authorization()

	if err := userService.LoadByID(builder.session(), authorization.UserID, &user); err != nil {
		return step.error(builder, "Invalid user. Sign out and try again.")
	}

	// Collect password from the Form POST
	transaction, err := formdata.Parse(builder.request())

	if err != nil {
		return step.error(builder, "Invalid form data. Reload and try again.")
	}

	password := transaction.Get("confirm_password")

	// Validate the password. If no match, then halt
	steranko := factory.Steranko(builder.session())
	if matches, _ := steranko.ComparePassword(password, user.GetPassword()); !matches {
		return step.error(builder, "Invalid password.")
	}

	// Success.  Allow the rest of the pipeline to continue.
	return nil
}

func (step StepRequirePassword) error(builder Builder, message string) PipelineBehavior {
	const location = "build.StepRequirePassword.Post"

	response := builder.response()
	response.Header().Set("HX-Reswap", "innerHTML")
	response.Header().Set("HX-Retarget", "#htmx-response-message")
	response.WriteHeader(http.StatusOK)

	if _, writeError := response.Write([]byte(`<span class="text-red">` + message + `</span>`)); writeError != nil {
		derp.Report(derp.Wrap(writeError, location, "Unable to write error message to response"))
	}

	return Halt().AsFullPage()
}
