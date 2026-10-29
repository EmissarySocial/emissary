package markdown

import (
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/schema/format"
)

// init registers SchemaFormat with rosetta.  rosetta freezes its format registry at the first
// schema lookup and requires registration during init, so this cannot wait for main.
func init() {
	schema.UseFormat(SchemaFormat, schemaFormat)
}

// schemaFormat returns a rosetta string format that sanitizes HTML with this package's policy,
// which keeps classes and iframes that rosetta's own "html" format removes.
func schemaFormat(_ string) format.StringFormat {
	return func(value string) (string, error) {
		return Sanitize(value), nil
	}
}
