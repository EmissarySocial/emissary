package build

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

// StepInlineSaveButton.Post replaces the button the visitor clicked, so the replacement must
// reproduce everything the original inherited from its surroundings -- most importantly its
// form association, which a button rendered outside its <form> has no other way to get.

// mustParse compiles a step-argument template, or fails the test
func mustParse(t *testing.T, source string) *template.Template {
	result, err := template.New("").Parse(source)
	require.Nil(t, err)
	return result
}

// TestStepInlineSaveButton_Post verifies the attributes on the replacement button
func TestStepInlineSaveButton_Post(t *testing.T) {

	testCases := []struct {
		name     string
		form     string
		expected []string
		absent   []string
	}{
		{
			name:     "form association",
			form:     "edit-form",
			expected: []string{`id="inline-save-button"`, `type="submit"`, `form="edit-form"`, `class="primary success"`, "Save Changes"},
		},
		{
			// A button inside its own <form> needs no association, and Attr writes no empty value
			name:     "no form",
			form:     "",
			expected: []string{`type="submit"`},
			absent:   []string{"form="},
		},
	}

	for _, testCase := range testCases {

		step := StepInlineSaveButton{
			ID:    mustParse(t, "inline-save-button"),
			Class: "primary",
			Label: mustParse(t, "Save Changes"),
			Form:  testCase.form,
		}

		var buffer bytes.Buffer
		result := step.Post(nil, &buffer)

		require.NotNil(t, result, testCase.name)

		for _, expected := range testCase.expected {
			require.Contains(t, buffer.String(), expected, testCase.name)
		}

		for _, absent := range testCase.absent {
			require.NotContains(t, buffer.String(), absent, testCase.name)
		}
	}
}

// TestStepInlineSaveButton_Retarget verifies that the response is aimed back at the button itself
func TestStepInlineSaveButton_Retarget(t *testing.T) {

	step := StepInlineSaveButton{
		ID:    mustParse(t, "my-button"),
		Class: "primary",
		Label: mustParse(t, "Save"),
		Form:  "edit-form",
	}

	var buffer bytes.Buffer
	status := NewPipelineResult()
	step.Post(nil, &buffer)(&status)

	// Halt is what lets "refresh-page" compose with this step -- it must run BEFORE this one
	require.True(t, status.Halt)
	require.Equal(t, "outerHTML", status.Headers["HX-Reswap"])
	require.Equal(t, "#my-button", status.Headers["HX-Retarget"])
}
