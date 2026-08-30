package step

import (
	"slices"
	"text/template"

	"github.com/benpate/rosetta/mapof"
)

// SendEmail is a Step that sends an email, either one of the built-in User emails
// or any email definition named by the Template
type SendEmail struct {
	Email  string                        // Name of the email to send
	Values map[string]*template.Template // Values passed into the email's data map
}

// NewSendEmail returns a fully initialized SendEmail object
func NewSendEmail(stepInfo mapof.Any) (SendEmail, error) {

	// Parse each value template
	valuesMap := stepInfo.GetMap("values")
	values := make(map[string]*template.Template, len(valuesMap))

	for key := range valuesMap {
		valueTemplate, err := template.New(key).Funcs(FuncMap()).Parse(valuesMap.GetString(key))

		if err != nil {
			return SendEmail{}, err
		}

		values[key] = valueTemplate
	}

	return SendEmail{
		Email:  stepInfo.GetString("email"),
		Values: values,
	}, nil
}

// builtInEmails names the emails that the User service sends itself, because each one mints a
// password-reset credential before it sends.  Neither has an email definition of its own.
var builtInEmails = []string{"welcome", "password-reset"}

// IsBuiltInEmail returns TRUE if emailID names one of the emails that the User service sends.
// Both the parse side and the execute side of "send-email" branch on this, so the list lives in
// one place -- two copies would let load-time validation and run-time dispatch disagree about
// which names are special, and a name known to only one of them fails at send time.
func IsBuiltInEmail(emailID string) bool {
	return slices.Contains(builtInEmails, emailID)
}

// EmailID returns the name of the email definition this step sends, or empty for a built-in.
// Load-time validation uses this to confirm the definition exists.
func (step SendEmail) EmailID() string {

	if IsBuiltInEmail(step.Email) {
		return ""
	}

	return step.Email
}

// Name returns the name of the step, which is used in debugging.
func (step SendEmail) Name() string {
	return "send-email"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SendEmail) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SendEmail) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SendEmail) RequiredRoles() []string {
	return []string{}
}
