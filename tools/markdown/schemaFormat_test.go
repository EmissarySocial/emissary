package markdown

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestSchemaFormat_Registered confirms that init registered SchemaFormat, so a schema
// that names it passes rosetta's load-time format check.
func TestSchemaFormat_Registered(t *testing.T) {

	element := schema.Object{Properties: schema.ElementMap{
		"html": schema.String{Format: SchemaFormat},
	}}

	require.NoError(t, schema.New(element).ValidateFormats())
}

// TestSchemaFormat_KeepsWhatTheRendererKeeps confirms that the format applies this
// package's policy: classes and iframes survive, and scripts and handlers do not.
func TestSchemaFormat_KeepsWhatTheRendererKeeps(t *testing.T) {

	sanitize := schemaFormat("")

	result, err := sanitize(`<pre><code class="language-go">x</code></pre><iframe src="https://example.com/v"></iframe><script>alert(1)</script><img src="x" onerror="alert(2)">`)

	require.NoError(t, err)
	require.Contains(t, result, `<code class="language-go">`)
	require.Contains(t, result, `<iframe src="https://example.com/v">`)
	require.NotContains(t, result, "<script")
	require.NotContains(t, result, "onerror")
}

// TestSchemaFormat_MatchesSanitize confirms that the format and Sanitize never disagree,
// since the schema gate exists to re-apply the same policy.
func TestSchemaFormat_MatchesSanitize(t *testing.T) {

	input := ToHTML("# Title\n\n```go\nx := 1\n```\n\n<iframe src=\"https://example.com/v\"></iframe>\n")

	result, err := schemaFormat("")(input)

	require.NoError(t, err)
	require.Equal(t, Sanitize(input), result)
}
