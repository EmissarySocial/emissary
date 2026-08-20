package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/slice"
)

// ThemeLookupProvider lists this server's Themes as form lookup codes
type ThemeLookupProvider struct {
	themeService *Theme
}

// NewThemeLookupProvider returns a fully initialized ThemeLookupProvider
func NewThemeLookupProvider(themeService *Theme) ThemeLookupProvider {
	return ThemeLookupProvider{
		themeService: themeService,
	}
}

// Get returns every active Theme. Implements the form.LookupGroup interface.
func (service ThemeLookupProvider) Get() []form.LookupCode {

	// Generate a slice containing all themes
	list := service.themeService.ListActive()

	// Convert the slice to a slice of LookupCodes
	return slice.Map(list, form.AsLookupCode[model.Theme])
}
