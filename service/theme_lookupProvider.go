package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/slice"
)

type ThemeLookupProvider struct {
	themeService *Theme
}

func NewThemeLookupProvider(themeService *Theme) ThemeLookupProvider {
	return ThemeLookupProvider{
		themeService: themeService,
	}
}

func (service ThemeLookupProvider) Get() []form.LookupCode {

	// Generate a slice containing all themes
	list := service.themeService.ListActive()

	// Convert the slice to a slice of LookupCodes
	return slice.Map(list, form.AsLookupCode[model.Theme])
}
