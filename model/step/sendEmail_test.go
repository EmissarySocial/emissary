package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSendEmail verifies that a "send-email" step parses its configuration
func TestSendEmail(t *testing.T) {
	step, err := NewSendEmail(mapof.Any{"email": "welcome"})
	require.Nil(t, err)
	require.Equal(t, "welcome", step.Email)

	require.Equal(t, "send-email", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSendEmail_TemplateEmail verifies that a step naming a Template-defined email parses its
// values, and reports the email ID for the load-time check that the definition exists
func TestSendEmail_TemplateEmail(t *testing.T) {

	step, err := NewSendEmail(mapof.Any{
		"email": "stream-contact-form",
		"values": mapof.Any{
			"To":         "{{.Data \"emailAddress\"}}",
			"ReplyEmail": "{{.GetString \"email\"}}",
		},
	})

	require.NoError(t, err)
	require.False(t, IsBuiltInEmail(step.Email))
	require.Equal(t, "stream-contact-form", step.EmailID())
	require.Len(t, step.Values, 2)
}

// TestIsBuiltInEmail verifies the one list that decides which email names the User service sends.
// Both the load-time check here and the run-time dispatch in build/step_SendEmail.go read it, so a
// name added to this list becomes special in both places at once, or in neither.
func TestIsBuiltInEmail(t *testing.T) {

	require.True(t, IsBuiltInEmail("welcome"))
	require.True(t, IsBuiltInEmail("password-reset"))

	require.False(t, IsBuiltInEmail("stream-contact-form"))
	require.False(t, IsBuiltInEmail("user-welcome"))
	require.False(t, IsBuiltInEmail(""))
}

// TestSendEmail_BuiltInsAreNotValidated verifies that the two built-in names report no email ID,
// so the load-time "does this definition exist" check skips them. They are sent through the User
// service, which mints a credential first, and never name a definition of their own.
func TestSendEmail_BuiltInsAreNotValidated(t *testing.T) {

	for _, name := range builtInEmails {

		step, err := NewSendEmail(mapof.Any{"email": name})

		require.NoError(t, err)
		require.Equal(t, "", step.EmailID())
	}
}
