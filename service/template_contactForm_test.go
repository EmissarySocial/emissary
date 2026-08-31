package service

import (
	"os"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Contact Form Template
 *
 * The contact form is the first Template that names an email
 * definition and supplies its data, and the first whose action
 * runs for an anonymous visitor. These tests pin both seams,
 * because neither is covered by the whole-tree template tests:
 * those unmarshal a definition, while the checks below are what
 * the Template service performs after every location has loaded.
 ******************************************/

// contactFormTemplate parses the shipped contact-form definition
func contactFormTemplate(t *testing.T) model.Template {

	t.Helper()

	definition, err := os.ReadFile("../_embed/templates/stream-contact-form/template.hjson")
	require.NoError(t, err)

	result := model.NewTemplate("contact-form", nil)
	require.NoError(t, hjson.Unmarshal(definition, &result))

	return result
}

// sendEmailStep returns the one send-email step in the named action
func sendEmailStep(t *testing.T, template model.Template, actionID string) modelStep.SendEmail {

	t.Helper()

	action, exists := template.Actions[actionID]
	require.True(t, exists, "template has no %q action", actionID)

	for _, step := range action.Steps {
		if emailStep, ok := step.(modelStep.SendEmail); ok {
			return emailStep
		}
	}

	require.FailNow(t, "action "+actionID+" has no send-email step")
	return modelStep.SendEmail{}
}

// TestContactFormTemplate_SuppliesEveryRequiredKey runs the same cross-check that
// Template.validateTemplates performs at load: every key the email's "to", "subject", and
// "headers" templates interpolate must appear in the step's values.  Those templates reject a
// missing key rather than rendering a blank, so a gap here is a send that dies at the moment a
// visitor presses the button.  Pinning it directly means a mismatch names its own cause.
func TestContactFormTemplate_SuppliesEveryRequiredKey(t *testing.T) {

	emailService, _ := contactFormEmail(t)
	step := sendEmailStep(t, contactFormTemplate(t), "submit")

	require.Equal(t, "contact-form", step.EmailID())

	required := emailService.RequiredKeys("contact-form")
	require.NotEmpty(t, required, "a contract with no keys would pass forever")

	for _, key := range required {
		require.Contains(t, step.Values, key,
			"the submit action does not supply %q, which the email's to/subject/headers templates require", key)
	}
}

// TestContactFormTemplate_SuppliesEveryBodyKey covers what load-time validation structurally
// cannot: body.html is html/template, which renders a missing key as "", so a key the body needs
// but the step omits produces a blank line in the recipient's email and no error anywhere.
func TestContactFormTemplate_SuppliesEveryBodyKey(t *testing.T) {

	step := sendEmailStep(t, contactFormTemplate(t), "submit")

	// Every key body.html interpolates, minus the Domain_* values that DomainEmail.Send injects
	for _, key := range []string{
		"Name", "ReplyEmail", "Message", "HeaderMessage",
		"Client_IP", "Client_Description", "Client_Referer", "Client_UserAgent",
		"Client_Brands", "Client_Platform", "Client_Mobile", "Client_AcceptLanguage",
		"Client_Accept", "Client_AcceptEncoding", "Client_DoNotTrack", "Client_PrivacyControl",
	} {
		require.Contains(t, step.Values, key, "body.html renders %q, which the submit action does not supply", key)
	}
}

// TestContactFormTemplate_ClientValuesAreNotFormFields verifies that no Client_* value can be
// posted.  These describe the browser, and their whole worth is that a sender cannot choose them:
// a Client_IP the visitor typed is worse than none, because it looks authoritative in the footer.
// read-form is an allowlist reading the body only (D23), so the guarantee is structural -- this
// pins it against a future edit that "helpfully" adds one of these names to the schema.
func TestContactFormTemplate_ClientValuesAreNotFormFields(t *testing.T) {

	action, exists := contactFormTemplate(t).Actions["submit"]
	require.True(t, exists)

	var readForm modelStep.ReadForm

	for _, step := range action.Steps {
		if typed, ok := step.(modelStep.ReadForm); ok {
			readForm = typed
		}
	}

	// The visitor declares exactly three fields, and none of them feeds a Client_* value
	for _, field := range []string{"clientIP", "ip", "ipAddress", "userAgent", "referer", "platform"} {
		_, exists := readForm.Schema.GetElement(field)
		require.False(t, exists, "read-form declares %q, which would let a visitor forge their own sender details", field)
	}
}

// TestContactFormTemplate_ClientValuesComeFromTheBuilder verifies that every Client_* value is
// rendered from a Builder accessor rather than from the transient form scope.  ".GetString" reads
// what read-form stored -- visitor input -- while the accessors read the request itself.  A step
// that mixed them would silently publish a forged fingerprint as fact.
func TestContactFormTemplate_ClientValuesComeFromTheBuilder(t *testing.T) {

	step := sendEmailStep(t, contactFormTemplate(t), "submit")

	for key, valueTemplate := range step.Values {

		if !strings.HasPrefix(key, "Client_") {
			continue
		}

		require.NotNil(t, valueTemplate.Tree, "%q must be a parsed template", key)
		require.NotContains(t, valueTemplate.Tree.Root.String(), "GetString",
			"%q reads visitor input; sender details must come from the request", key)
		require.NotContains(t, valueTemplate.Tree.Root.String(), "QueryParam",
			"%q reads the query string, which the visitor also controls", key)
	}
}

// TestContactFormTemplate_ReadFormDeclaresEveryVisitorField verifies that every visitor value the
// send-email step reads out of the transient scope was first declared to read-form.  The read-form
// schema is an allowlist, so a field missing from it is never read and reaches the email empty.
func TestContactFormTemplate_ReadFormDeclaresEveryVisitorField(t *testing.T) {

	action, exists := contactFormTemplate(t).Actions["submit"]
	require.True(t, exists)

	var readForm modelStep.ReadForm
	var found bool

	for _, step := range action.Steps {
		if typed, ok := step.(modelStep.ReadForm); ok {
			readForm, found = typed, true
		}
	}

	require.True(t, found, "the submit action must read the form before it sends")

	for _, field := range []string{"name", "email", "message"} {
		element, exists := readForm.Schema.GetElement(field)
		require.True(t, exists, "read-form does not declare %q, so the visitor's value is dropped", field)
		require.NotNil(t, element)
	}
}

// TestContactFormTemplate_SubmitIsScopedToPublished verifies the authorization that the whole
// anonymous pipeline rests on: a published form accepts a visitor, an unpublished one does not.
// An anonymous grant is still scoped by state, and this is what enforces that -- without it, a
// draft contact form would email its author's inbox from the moment it was created.
func TestContactFormTemplate_SubmitIsScopedToPublished(t *testing.T) {

	template := contactFormTemplate(t)
	permissionService := NewPermission()
	authorization := model.NewAuthorization()

	action := template.Actions["submit"]
	require.NoError(t, action.CalcAccessList(&template, false))
	template.Actions["submit"] = action

	// Both Streams carry a real author.  A zero AttributedTo.UserID would equal
	// MagicGroupIDAnonymous and open the author-gated draft to everyone -- see
	// TestStream_RolesToGroupIDs_ZeroAuthorIsNotAnonymous.
	author := primitive.NewObjectID()

	// A published form admits an anonymous visitor
	published := model.NewStream()
	published.StateID = "published"
	published.AttributedTo = model.PersonLink{UserID: author}

	allowed, err := permissionService.UserCan(nil, &authorization, &template, &published, "submit")
	require.Nil(t, err)
	require.True(t, allowed, "a published contact form must accept an anonymous visitor")

	// The same action on an unpublished form does not
	draft := model.NewStream()
	draft.StateID = "default"
	draft.AttributedTo = model.PersonLink{UserID: author}

	allowed, err = permissionService.UserCan(nil, &authorization, &template, &draft, "submit")
	require.Nil(t, err)
	require.False(t, allowed, "an unpublished contact form must not accept messages")
}

// TestContactFormTemplate_ViewMatchesSubmit verifies that the form is submittable exactly when the
// page is viewable.  A visitor who can see the form and cannot use it, or the reverse, is a bug in
// either direction, and the two access lists are declared separately.
func TestContactFormTemplate_ViewMatchesSubmit(t *testing.T) {

	template := contactFormTemplate(t)

	for _, actionID := range []string{"view", "submit"} {
		action := template.Actions[actionID]
		require.NoError(t, action.CalcAccessList(&template, false))
		template.Actions[actionID] = action
	}

	view := template.Actions["view"].AccessList["published"]
	submit := template.Actions["submit"].AccessList["published"]

	require.Equal(t, []string{model.MagicRoleAnonymous}, []string(view))
	require.Equal(t, []string(view), []string(submit), "view and submit must open to the same audience")
}
