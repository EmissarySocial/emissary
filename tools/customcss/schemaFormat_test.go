package customcss

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// stylesheetSchema returns a schema with one stylesheet field in the given format
func stylesheetSchema(format string) schema.Schema {
	return schema.New(schema.Object{
		Properties: schema.ElementMap{
			"stylesheet": schema.String{Format: format},
		},
	})
}

// TestSchemaFormat_Registered confirms that init registered SchemaFormat, so a schema that
// names it, with or without its argument, passes rosetta's load-time format check
func TestSchemaFormat_Registered(t *testing.T) {
	require.NoError(t, stylesheetSchema(SchemaFormat).ValidateFormats())
	require.NoError(t, stylesheetSchema(SchemaFormat+"="+SchemaArgumentRemote).ValidateFormats())
}

// TestSchemaFormat_KeepsTheTextAsWritten confirms that an accepted stylesheet is stored byte for byte
func TestSchemaFormat_KeepsTheTextAsWritten(t *testing.T) {

	value := "/* Profile */\n#profile-card {\n\tborder: 2px solid red;\n}\n\nnav > a::before { content: \"<b>\"; }\n"
	object := mapof.NewAny()

	require.NoError(t, stylesheetSchema(SchemaFormat).Set(&object, "stylesheet", value))
	require.Equal(t, value, object["stylesheet"])
}

// TestSchemaFormat_RefusesWithTheRuleItBreaks confirms that a refusal reaches the caller as a
// validation error whose root message is the one Validate wrote, after rosetta's field path
func TestSchemaFormat_RefusesWithTheRuleItBreaks(t *testing.T) {

	object := mapof.NewAny()
	err := stylesheetSchema(SchemaFormat).Set(&object, "stylesheet", "a {\n\tbackground: url(https://example.com/a.png);\n}")

	require.Error(t, err)
	require.True(t, derp.IsValidationError(err))
	require.Equal(t, "stylesheet: The stylesheet cannot load files from other websites (line 2).", derp.RootMessage(err))
	require.Empty(t, object["stylesheet"])
}

// TestSchemaFormat_Remote confirms that the remote argument relaxes only the rule against other websites
func TestSchemaFormat_Remote(t *testing.T) {

	remote := schemaFormat(SchemaArgumentRemote)

	value, err := remote("@import url(https://fonts.example/x.css);")
	require.NoError(t, err)
	require.Equal(t, "@import url(https://fonts.example/x.css);", value)

	value, err = remote("a {} }")
	require.Error(t, err)
	require.Empty(t, value)
}

// TestSchemaFormat_UnknownArgumentIsStrict confirms that a misspelled argument keeps every rule
func TestSchemaFormat_UnknownArgumentIsStrict(t *testing.T) {

	_, err := schemaFormat("remot")("@import url(https://fonts.example/x.css);")
	require.Error(t, err)
}
