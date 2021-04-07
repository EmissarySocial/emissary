package vocabulary

import (
	"testing"

	"github.com/benpate/form"
	"github.com/stretchr/testify/assert"
)

func TestTextarea(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "textarea",
		Path: "username",
	}

	html, err := f.HTML(library, s, nil)

	assert.Nil(t, err)
	assert.Equal(t, `<textarea name="username" minlength="10" maxlength="100" pattern="[a-z]+" required="true"></textarea>`, html)
}
