package service

import (
	"sort"

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
	list := service.themeService.List()

	// Sort the slice by rank
	sort.Slice(list, func(i int, j int) bool {
		return list[i].Rank > list[j].Rank
	})

	// Convert the slice to a slice of LookupCodes
	return slice.Map(list, func(item model.Theme) form.LookupCode {
		return item.LookupCode()
	})
}
