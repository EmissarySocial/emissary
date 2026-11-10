package dataset

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/icon/bootstrap"
	"github.com/benpate/toot/scope"
	"github.com/stretchr/testify/require"
)

// oauthScopeValues is every scope the dataset must describe: each Mastodon scope that toot defines, the two that
// it does not, and Emissary's own.  A new toot release that adds a scope needs a row here and in the dataset.
var oauthScopeValues = []string{
	model.OAuthUserTokenScopeProfile,
	model.OAuthUserTokenScopeEmail,
	scope.Read, scope.ReadAccounts, scope.ReadBlocks, scope.ReadBookmarks, scope.ReadCollections,
	scope.ReadFavourites, scope.ReadFilters, scope.ReadFollows, scope.ReadLists, scope.ReadMutes,
	scope.ReadNotifications, scope.ReadSearch, scope.ReadStatuses, scope.Push,
	scope.Write, scope.WriteAccounts, scope.WriteBlocks, scope.WriteBookmarks, scope.WriteCollections,
	scope.WriteConversations, scope.WriteFavourites, scope.WriteFilters, scope.WriteFollows, scope.WriteLists,
	scope.WriteMedia, scope.WriteMutes, scope.WriteNotifications, scope.WriteReports, scope.WriteStatuses,
	"follow",
	scope.AdminRead, scope.AdminReadAccounts, scope.AdminReadReports, scope.AdminReadDomainAllows,
	scope.AdminReadDomainBlocks, scope.AdminReadIPBlocks, scope.AdminReadEmailDomainBlocks,
	scope.AdminReadCanonicalEmailBlocks,
	scope.AdminWrite, scope.AdminWriteAccounts, scope.AdminWriteReports, scope.AdminWriteDomainAllows,
	scope.AdminWriteDomainBlocks, scope.AdminWriteIPBlocks, scope.AdminWriteEmailDomainBlocks,
	scope.AdminWriteCanonicalEmailBlocks,
	model.OAuthUserTokenScopeActivityPubPortability,
}

// TestOAuthScopes_DescribesEveryScopeInOrder pins the whole set, in the order the consent page lists it
func TestOAuthScopes_DescribesEveryScopeInOrder(t *testing.T) {

	values := make([]string, 0, len(oauthScopeValues))

	for _, lookupCode := range OAuthScopes() {
		values = append(values, lookupCode.Value)
	}

	require.Equal(t, oauthScopeValues, values)
}

// TestOAuthScopes_EveryCodeIsComplete requires the words and the icon that the consent page draws for every scope
func TestOAuthScopes_EveryCodeIsComplete(t *testing.T) {

	for _, lookupCode := range OAuthScopes() {
		require.NotEmpty(t, lookupCode.Label, "scope %q has no Label", lookupCode.Value)
		require.NotEmpty(t, lookupCode.Description, "scope %q has no Description", lookupCode.Value)
		require.NotEmpty(t, lookupCode.Icon, "scope %q has no Icon", lookupCode.Value)
		require.NotEmpty(t, lookupCode.Group, "scope %q has no Group", lookupCode.Value)
		require.Empty(t, lookupCode.Href, "scope %q links nowhere", lookupCode.Value)
	}
}

// TestOAuthScopes_GroupsAreContiguousAndOrdered pins the group order, because groupie draws one header per run
func TestOAuthScopes_GroupsAreContiguousAndOrdered(t *testing.T) {

	seen := make([]string, 0)

	for _, lookupCode := range OAuthScopes() {

		if (len(seen) == 0) || (seen[len(seen)-1] != lookupCode.Group) {
			seen = append(seen, lookupCode.Group)
		}
	}

	require.Equal(t, []string{"Identity", "Read Access", "Write Access", "Moderation", "Data Portability"}, seen)
}

// TestOAuthScopes_EveryIconDrawsItself requires each icon to be in the vendored set, under its own name
func TestOAuthScopes_EveryIconDrawsItself(t *testing.T) {

	available := availableIcons(t)
	provider := bootstrap.Provider{}

	for _, lookupCode := range OAuthScopes() {

		_, exists := available[lookupCode.Icon]
		require.True(t, exists, "scope %q draws %q, which is not in Bootstrap Icons 1.13.1", lookupCode.Value, lookupCode.Icon)

		matches := iconClassPattern.FindStringSubmatch(provider.Get(lookupCode.Icon))
		require.Len(t, matches, 2, "provider returned unparsable markup for %q", lookupCode.Icon)
		require.Equal(t, lookupCode.Icon, matches[1], "scope %q names %q, but the provider draws %q", lookupCode.Value, lookupCode.Icon, matches[1])
	}
}

// TestOAuthScopes_CallersCannotCorruptTheList proves that each call returns a list of its own
func TestOAuthScopes_CallersCannotCorruptTheList(t *testing.T) {

	first := OAuthScopes()
	first[0].Label = "changed"

	require.Equal(t, "Your Profile", OAuthScopes()[0].Label)
}
