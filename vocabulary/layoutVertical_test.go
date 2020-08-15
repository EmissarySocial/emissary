package vocabulary

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/stretchr/testify/assert"
)

func TestLayoutVertical(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "layout-vertical",
		Children: []form.Form{
			{
				Kind:  "text",
				Label: "Name",
				Path:  "name",
			},
			{
				Kind:  "text",
				Label: "Email",
				Path:  "email",
			},
			{
				Kind:  "text",
				Label: "Age",
				Path:  "age",
			},
		},
	}

	v := map[string]interface{}{
		"name":  "John Connor",
		"email": "john@resistance.mil",
		"age":   27,
	}

	html, err := f.HTML(library, s, v)

	assert.Nil(t, err)
	derp.Report(err)

	t.Log(html)
}
