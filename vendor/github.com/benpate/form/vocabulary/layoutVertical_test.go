package vocabulary

import (
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayoutVertical(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind:  "layout-vertical",
		Label: "This is my Vertical Layout",
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

	expected := `
	<div class="layout-vertical">
		<div class="layout-vertical-label">This is my Vertical Layout</div>
		<div class="layout-vertical-elements">
			<div class="layout-vertical-element">
				<label>Name</label>
				<input name="name" value="John Connor" type="text" maxlength="50">
			</div>
			<div class="layout-vertical-element">
				<label>Email</label>
				<input name="email" value="john@resistance.mil" type="email" minlength="10" maxlength="100" required="true">
			</div>
			<div class="layout-vertical-element">
				<label>Age</label>
				<input name="age" value="27" type="number" step="1" min="10" max="100" required="true">
			</div>
		</div>
	</div>`

	expected = strings.ReplaceAll(expected, "\n", "")
	expected = strings.ReplaceAll(expected, "\t", "")
	require.Equal(t, expected, html)
}
