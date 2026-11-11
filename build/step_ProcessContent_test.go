package build

import (
	"testing"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestStepProcessContent_useHardWraps verifies that only MARKDOWN content whose template executes to "true" is hard-wrapped
func TestStepProcessContent_useHardWraps(t *testing.T) {

	data := map[string]any{"yes": true, "no": false, "text": "sometimes"}

	tests := []struct {
		name      string
		format    string
		hardWraps string
		expected  bool
	}{
		{name: "literal true", format: model.ContentFormatMarkdown, hardWraps: "true", expected: true},
		{name: "literal false", format: model.ContentFormatMarkdown, hardWraps: "false", expected: false},
		{name: "template true", format: model.ContentFormatMarkdown, hardWraps: "{{.yes}}", expected: true},
		{name: "template false", format: model.ContentFormatMarkdown, hardWraps: "{{.no}}", expected: false},
		{name: "padded true", format: model.ContentFormatMarkdown, hardWraps: " {{.yes}}\n", expected: true},
		{name: "missing value", format: model.ContentFormatMarkdown, hardWraps: "{{.absent}}", expected: false},
		{name: "any other text", format: model.ContentFormatMarkdown, hardWraps: "{{.text}}", expected: false},
		{name: "empty", format: model.ContentFormatMarkdown, hardWraps: "", expected: false},
		{name: "not markdown", format: model.ContentFormatHTML, hardWraps: "true", expected: false},
		{name: "no format", format: "", hardWraps: "true", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step := StepProcessContent{
				Format:    test.format,
				HardWraps: template.Must(template.New("hard-wraps").Parse(test.hardWraps)),
			}

			require.Equal(t, test.expected, step.useHardWraps(data))
		})
	}
}

// TestStepProcessContent_useHardWraps_NoTemplate verifies that a step built without a template keeps standard Markdown
func TestStepProcessContent_useHardWraps_NoTemplate(t *testing.T) {
	step := StepProcessContent{Format: model.ContentFormatMarkdown}
	require.False(t, step.useHardWraps(nil))
}
