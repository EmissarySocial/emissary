package vocabulary

import (
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/null"
	"github.com/benpate/schema"
	"github.com/stretchr/testify/assert"
)

func getTestLibrary() form.Library {

	library := form.New()

	All(library)

	return library
}

func getTestSchema() schema.Schema {

	return schema.Schema{
		ID:      "",
		Comment: "",
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"username": schema.String{
					MinLength: null.NewInt(10),
					MaxLength: null.NewInt(100),
					Pattern:   "[a-z]+",
					Required:  true,
				},
				"name": schema.String{
					MaxLength: null.NewInt(50),
				},
				"email": schema.String{
					Format:    "email",
					MinLength: null.NewInt(10),
					MaxLength: null.NewInt(100),
					Required:  true,
				},
				"age": schema.Integer{
					Minimum: null.NewInt(10),
					Maximum: null.NewInt(100),
					// Required: true,
				},
				"distance": schema.Number{
					Minimum: null.NewFloat(10),
					Maximum: null.NewFloat(100),
					// Required: true,
				},
			},
		},
	}

}
func TestAll(t *testing.T) {

	library := getTestLibrary()

	assert.NotNil(t, library)
}
