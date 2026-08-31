package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// stubEmailFactory satisfies the Factory interface by embedding it (nil) and overriding only
// Email().  It hands back a zero-value DomainEmail, whose SMTP connection is unconfigured --
// which is a real production state, not a mock: a domain whose owner has not filled in SMTP
// yet.  DomainEmail.Send deliberately treats that as a failure rather than a silent success,
// so it is also the cheapest way to exercise this step's error path end to end.
type stubEmailFactory struct {
	Factory
	emailService *service.DomainEmail
}

// Email returns the stubbed email service. Implements the Factory interface.
func (factory stubEmailFactory) Email() *service.DomainEmail {
	return factory.emailService
}

// newSendEmailBuilder assembles a Stream builder with an email service attached
func newSendEmailBuilder(t *testing.T) Stream {
	t.Helper()

	emailService := service.NewDomainEmail()
	stream := model.NewStream()
	stream.Label = "Contact Us"

	template := model.NewTemplate("test", nil)

	return Stream{
		_stream: &stream,
		CommonWithTemplate: CommonWithTemplate{
			_template: template,
			Common: Common{
				_factory: stubEmailFactory{emailService: &emailService},
				_request: httptest.NewRequest(http.MethodPost, "/000000000000000000000000/submit", nil),
			},
		},
	}
}

// mustParseValue compiles a step-argument template, or fails the test
func mustParseValue(t *testing.T, source string) *template.Template {
	t.Helper()
	result, err := template.New("").Parse(source)
	require.Nil(t, err)
	return result
}

// TestStepSendEmail_Get proves that a GET never sends mail.
//
// Every action's pipeline runs on both verbs unless a step opts out, and this step opts out by
// doing nothing on GET.  Without that, merely rendering a page that contains a send-email step
// would deliver a message -- and link prefetchers and mail scanners issue GETs unprompted.
func TestStepSendEmail_Get(t *testing.T) {

	step := StepSendEmail{Email: "contact-form"}

	require.Nil(t, step.Get(newSendEmailBuilder(t), io.Discard))
}

// TestStepSendEmail_PostHaltsOnFailure proves that a failed send stops the pipeline.
//
// This step is the one place in a web-form pipeline where the message exists only in flight:
// nothing is written to the database and nothing is queued.  An error reported-and-continued
// would therefore return a success page to a visitor whose message reached nobody, and leave
// no record that it ever existed.
func TestStepSendEmail_PostHaltsOnFailure(t *testing.T) {

	step := StepSendEmail{
		Email: "contact-form",
		Values: map[string]*template.Template{
			"To":      mustParseValue(t, `{{.Data "emailAddress"}}`),
			"Subject": mustParseValue(t, `{{.Label}}`),
		},
	}

	result := applyBehavior(step.Post(newSendEmailBuilder(t), io.Discard))

	require.True(t, result.Halt, "a send failure must halt the pipeline")
	require.NotNil(t, result.Error)
	require.Contains(t, derp.Message(result.Error), "Error sending email")
}

// TestStepSendEmail_BuiltInRequiresUserBuilder proves that the two built-in emails refuse to
// send from anything but a User builder.
//
// "welcome" and "password-reset" each mint a password-reset credential for the User the builder
// carries.  Reached from a Stream, there is no User to mint one for -- so this must fail loudly
// rather than fall through to the generic path, where it would look up an email definition that
// does not exist.
func TestStepSendEmail_BuiltInRequiresUserBuilder(t *testing.T) {

	for _, emailID := range []string{"welcome", "password-reset"} {

		step := StepSendEmail{Email: emailID}
		result := applyBehavior(step.Post(newSendEmailBuilder(t), io.Discard))

		require.True(t, result.Halt, "%s must halt on a non-User builder", emailID)
		require.NotNil(t, result.Error, "%s", emailID)
		require.Contains(t, derp.Message(result.Error), "Invalid Builder", "%s", emailID)
	}
}

// TestStepSendEmail_TemplateEmailUsesGenericPath proves that a name outside the built-in list
// routes to the Template-defined path, and NOT to the built-in one.
//
// The two paths differ in more than plumbing: the built-in path reports-and-continues, while
// the Template path halts.  Misrouting a name would silently swap a form's failure mode.
func TestStepSendEmail_TemplateEmailUsesGenericPath(t *testing.T) {

	step := StepSendEmail{Email: "contact-form"}
	result := applyBehavior(step.Post(newSendEmailBuilder(t), io.Discard))

	require.NotNil(t, result.Error)
	require.NotContains(t, derp.Message(result.Error), "Invalid Builder",
		"a Template-defined email must not route through the built-in path")
}
