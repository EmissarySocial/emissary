package customcss

import (
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/schema/format"
)

// init registers SchemaFormat with rosetta, which accepts a format only before its first schema lookup
func init() {
	schema.UseFormat(SchemaFormat, schemaFormat)
}

// schemaFormat returns a rosetta string format that keeps a stylesheet exactly as written, or refuses it
func schemaFormat(argument string) format.StringFormat {

	remoteAllowed := (argument == SchemaArgumentRemote)

	return func(value string) (string, error) {

		if err := validate(value, remoteAllowed); err != nil {
			return "", err
		}

		return value, nil
	}
}
