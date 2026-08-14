package model

import (
	"net/url"
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDomainSchema(t *testing.T) {

	domain := NewDomain()

	// The virtual iconUrl/imageUrl fields derive from Host() + attachment path, so a
	// hostname is required for them to pass the (absolute-only) "url" format.
	domain.Hostname = "example.com"

	s := schema.New(DomainSchema())

	table := []tableTestItem{
		{"domainId", "123456781234567812345678", nil},
		{"iconId", "aaa4bbb8ddd4ddd812345678", nil},
		{"themeId", "123456516253413243716253", nil},
		{"registrationId", "none", nil},
		{"inboxId", "user-inbox", nil},
		{"outboxId", "user-outbox", nil},
		{"registrationData.customA", "CUSTOM", nil},
		{"registrationData.customB", "CUSTOM", nil},
		{"registrationData.customC", "CUSTOM", nil},
		{"label", "LABEL", nil},
		{"description", "DESCRIPTION", nil},
		{"forward", "https://other.site", nil},
		{"data.custom", "CUSTOM", nil},
		{"data.value", "VALUE", nil},
		{"data.sso_active", "true", nil},
		{"data.sso_secret", "123456789-10-11-12", nil},
		{"colorMode", "LIGHT", nil},
		{"registrationData.custom", "CUSTOM", nil},
		{"registrationData.value", "VALUE", nil},
		{"syndication.0.value", "VALUE", nil},
		{"syndication.0.label", "LABEL", nil},
		{"syndication.1.description", "DESCRIPTION", nil},
		{"syndication.1.href", "https://syndication.site", nil},
		{"defaultAnonymous", "/home", nil},
		{"defaultAuthenticated", "/@me", nil},
		{"defaultOwner", "/admin", nil},
		{"imageId", "aaa4bbb8ddd4ddd812345679", nil},
		{"mlsGroupIds", "GROUP-IDS", nil},
		{"mlsMode", DomainMLSModeGroups, nil},
	}

	tableTest_Schema(t, &s, &domain, table)
}

// TestDomainURLs_RequireHostname pins the reason a Domain must carry its own hostname: Host()
// prefixes every derived URL, so a blank Hostname yields "https:///..." -- a scheme with no
// authority.  That value reaches federation actors, OAuth client metadata, oEmbed, and email links,
// and it is rejected outright by any schema that declares format:"url" on iconUrl (which
// theme-default does, and which service.Domain.persist validates on EVERY write).  The hostname is
// stamped in service.Domain, by bootstrap and by stampHostname on each Start.
func TestDomainURLs_RequireHostname(t *testing.T) {

	// urlFormat mirrors the constraint that theme-default/theme.hjson puts on iconUrl
	urlFormat := schema.New(schema.String{Format: "url"})

	requireAbsolute := func(t *testing.T, value string) {
		t.Helper()

		parsed, err := url.Parse(value)
		require.Nil(t, err)
		require.NotEmpty(t, parsed.Scheme, "derived URL must carry a scheme")
		require.NotEmpty(t, parsed.Host, "derived URL must carry an authority: %s", value)

		_, formatErr := urlFormat.Validate(value)
		require.Nil(t, formatErr, "derived URL must satisfy format:\"url\": %s", value)
	}

	t.Run("PublicHostnameUsesHTTPS", func(t *testing.T) {
		domain := NewDomain()
		domain.Hostname = "example.com"

		require.Equal(t, "https://example.com", domain.Host())
		requireAbsolute(t, domain.IconURL())
		requireAbsolute(t, domain.ImageURL())
	})

	t.Run("LocalHostnameUsesHTTP", func(t *testing.T) {
		domain := NewDomain()
		domain.Hostname = "localhost"

		require.Equal(t, "http://localhost", domain.Host())
		requireAbsolute(t, domain.IconURL())
		requireAbsolute(t, domain.ImageURL())
	})

	t.Run("UploadedArtwork", func(t *testing.T) {
		// Uploaded artwork takes the other branch of IconURL/ImageURL
		domain := NewDomain()
		domain.Hostname = "example.com"
		domain.IconID = primitive.NewObjectID()
		domain.ImageID = primitive.NewObjectID()

		requireAbsolute(t, domain.IconURL())
		requireAbsolute(t, domain.ImageURL())
	})

	t.Run("BlankHostnameProducesNoAuthority", func(t *testing.T) {
		// This is the failure the hostname stamp exists to prevent
		domain := NewDomain()

		require.Equal(t, "https://", domain.Host())

		parsed, err := url.Parse(domain.ImageURL())
		require.Nil(t, err)
		require.Empty(t, parsed.Host)

		_, formatErr := urlFormat.Validate(domain.IconURL())
		require.NotNil(t, formatErr)
	})
}
