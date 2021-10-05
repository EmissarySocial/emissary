package vocabulary

import (
	"testing"

	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
)

func TestSelectOne(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "select",
		Path: "color",
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	t.Log(html)
}

func TestSelectOneFromProvider(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "select",
		Path: "color",
		Options: map[string]string{
			"provider": "/test",
		},
	}

	value := map[string]interface{}{"color": "FIVE"}

	html, err := f.HTML(library, s, value)

	require.Nil(t, err)
	t.Log(html)
}

func TestSelectOneRadio(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "select",
		Path: "color",
		Options: map[string]string{
			"format": "radio",
		},
	}

	html, err := f.HTML(library, s, nil)

	require.Nil(t, err)
	t.Log(html)
}

func TestSelectMany(t *testing.T) {

	library := getTestLibrary()
	s := getTestSchema()

	f := form.Form{
		Kind: "select",
		Path: "tags",
	}

	value := map[string]interface{}{
		"tags": []string{"pretty", "please"},
	}

	html, err := f.HTML(library, s, value)

	require.Nil(t, err)
	t.Log(html)
}
