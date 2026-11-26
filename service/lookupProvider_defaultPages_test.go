package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
)

// lookupCodeValues returns the Value of every LookupCode, in order
func lookupCodeValues(lookupCodes []form.LookupCode) []string {

	result := make([]string, 0, len(lookupCodes))

	for _, lookupCode := range lookupCodes {
		result = append(result, lookupCode.Value)
	}

	return result
}

// TestDefaultPageCodes_OrdersThemeThenPagesThenBuiltIns verifies the order that the picker offers its pages in
func TestDefaultPageCodes_OrdersThemeThenPagesThenBuiltIns(t *testing.T) {

	pages := model.StartupDefaultPages{Anonymous: "/artists", Authenticated: "/@me", Owner: "/admin"}
	result := defaultPageCodes(pages, []string{"home", "artists", "albums"})

	require.Equal(t, []string{"/artists", "/@me", "/admin", "/home", "/albums", "/@me/newsfeed"}, lookupCodeValues(result))
}

// TestDefaultPageCodes_LabelsMatchValues verifies that each page is labeled with its own path
func TestDefaultPageCodes_LabelsMatchValues(t *testing.T) {

	for _, lookupCode := range defaultPageCodes(model.StartupDefaultPages{Anonymous: "/map"}, []string{"about"}) {
		require.Equal(t, lookupCode.Value, lookupCode.Label)
	}
}

// TestDefaultPageCodes_EmptyInputs verifies that a Theme with no landing pages and a Domain with no pages still offer the built-ins
func TestDefaultPageCodes_EmptyInputs(t *testing.T) {

	result := defaultPageCodes(model.StartupDefaultPages{}, nil)

	require.Equal(t, []string{"/home", "/@me", "/@me/newsfeed", "/admin"}, lookupCodeValues(result))
}

// TestDefaultPageCodes_SkipsEmptyTokens verifies that a Stream with no token offers no page
func TestDefaultPageCodes_SkipsEmptyTokens(t *testing.T) {

	result := defaultPageCodes(model.StartupDefaultPages{}, []string{"", "about", ""})

	require.Equal(t, []string{"/about", "/home", "/@me", "/@me/newsfeed", "/admin"}, lookupCodeValues(result))
}
