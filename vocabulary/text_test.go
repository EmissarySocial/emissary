package vocabulary

import (
	"testing"

	"github.com/benpate/form"
	"github.com/stretchr/testify/assert"
)

func TestInteger(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "age",
	}

	html, err := f.HTML(library, s, nil)

	assert.Nil(t, err)
	assert.Equal(t, `<input name="age" type="number" step="1" min="10" max="100" required="true">`, html)
}

func TestFloat(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		ID:   "idFormElement",
		Kind: "text",
		Path: "distance",
	}

	html, err := f.HTML(library, s, nil)

	assert.Nil(t, err)
	assert.Equal(t, `<input id="idFormElement" name="distance" type="number" min="10" max="100" required="true">`, html)
}

func TestText(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "username",
	}

	html, err := f.HTML(library, s, nil)

	assert.Nil(t, err)
	assert.Equal(t, `<input name="username" type="text" minlength="10" maxlength="100" pattern="[a-z]+" required="true">`, html)
}

func TestDescription(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind:        "text",
		Path:        "name",
		Description: "Hint text would go here",
	}

	html, err := f.HTML(library, s, nil)

	assert.Nil(t, err)
	assert.Equal(t, `<input name="name" type="text" maxlength="50" hint="Hint text would go here">`, html)
}
