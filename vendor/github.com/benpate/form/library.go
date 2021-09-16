package form

import (
	"github.com/benpate/derp"
	"github.com/benpate/schema"
)

// Library stores all of the available Renderers, and can execute them on a set of data
type Library struct {
	Provider  OptionProvider
	Renderers map[string]Renderer
}

// New returns a fully initialized Library
func New(provider OptionProvider) Library {
	return Library{
		Provider:  provider,
		Renderers: make(map[string]Renderer),
	}
}

// Register adds a new Renderer to the form.Library
func (library *Library) Register(name string, renderer Renderer) {
	library.Renderers[name] = renderer
}

// Renderer retrieves a renderer function from the library
func (library Library) Renderer(name string) (Renderer, error) {

	if renderer, ok := library.Renderers[name]; ok {
		return renderer, nil
	}

	return nil, derp.New(500, "form.Library.Renderer", "Undefined Renderer", name)
}

func (library Library) Options(form Form, element schema.Element) []OptionCode {

	// If form specifies an OptionProvider, then use that
	if optionProvider := form.Options["provider"]; optionProvider != "" {
		result, err := library.Provider.OptionCodes((optionProvider))

		if err != nil {
			derp.Report(err)
		}

		return result
	}

	// If this is an array, then look up Enumerations on its elements.
	if array, ok := element.(schema.Array); ok {
		element = array.Items
	}

	// If this schema element is an Enumerator, then convert its values to []OptionCode
	if enumerator, ok := element.(schema.Enumerator); ok {
		options := enumerator.Enumerate()

		result := make([]OptionCode, len(options))
		for index, value := range options {
			result[index] = OptionCode{Label: value, Value: value}
		}
		return result
	}

	// Fall through to "no options available"
	return make([]OptionCode, 0)
}
