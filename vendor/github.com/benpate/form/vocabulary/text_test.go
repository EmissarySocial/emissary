package vocabulary

import (
	"testing"

	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
)

func TestInteger(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "age",
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	require.Equal(t, `<input name="age" type="number" step="1" min="10" max="100" required="true">`, html)
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

	require.Nil(t, err)
	require.Equal(t, `<input id="idFormElement" name="distance" type="number" min="10" max="100" required="true">`, html)
}

func TestText(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "username",
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	require.Equal(t, `<input name="username" type="text" minlength="10" maxlength="100" pattern="[a-z]+" required="true">`, html)
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

	require.Nil(t, err)
	require.Equal(t, `<input name="name" type="text" maxlength="50" hint="Hint text would go here">`, html)
}

func TestTextTags(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "tags",
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	require.Equal(t, `<input name="tags" list="datalist_tags" type="text"><datalist id="datalist_tags"><option value="pretty"><option value="please"><option value="my"><option value="dear"><option value="aunt"><option value="sally"></datalist>`, html)
}

func TestTextTagsWithID(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "tags",
		ID:   "tags",
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	require.Equal(t, `<input id="tags" name="tags" list="datalist_tags" type="text"><datalist id="datalist_tags"><option value="pretty"><option value="please"><option value="my"><option value="dear"><option value="aunt"><option value="sally"></datalist>`, html)
}

func TestTextOptions(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "text",
		Path: "tag",
		ID:   "tag",
		Options: map[string]string{
			"provider": "/test",
		},
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	t.Log(html)
}
