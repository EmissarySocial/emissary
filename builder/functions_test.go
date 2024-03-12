package builder

import (
	"html/template"
	"testing"

	"github.com/benpate/icon/bootstrap"
	"github.com/stretchr/testify/require"
)

func TestFunctions_Icon(t *testing.T) {

	f := FuncMap(bootstrap.Provider{})

	icon := f["icon"].(func(string) template.HTML)

	require.Equal(t, template.HTML(`<i class="bi bi-check-lg"></i>`), icon("save"))
}

func TestFunctions_DollarFormat(t *testing.T) {

	f := FuncMap(bootstrap.Provider{})

	dollarFormat := f["dollarFormat"].(func(any) string)

	require.Equal(t, "$12.34", dollarFormat(1234))
}
