package model

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestDomainSchema returns the rosetta schema that describes a TestDomain
func TestDomainSchema(t *testing.T) {

	writableDomain := NewWritableDomain()

	// The virtual iconUrl/imageUrl fields derive from Host() + attachment path, so a
	// hostname is required for them to pass the (absolute-only) "url" format.
	writableDomain.Hostname = "example.com"

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
		{"themeData.custom", "CUSTOM", nil},
		{"themeData.stylesheet", "body { color: red; }", nil},
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

	tableTest_Schema(t, &s, &writableDomain, table)
}

// TestDomainSchema_ThemeDataIsSeparateFromData pins the separation between a Domain's two
// custom-value maps.  `themeData` holds the Theme's public values and is rendered into
// pages; `data` holds operational secrets (the VAPID private key, SSO secrets) and is not.
// Writing a theme value into `data` would stage those secrets for publication.
func TestDomainSchema_ThemeDataIsSeparateFromData(t *testing.T) {

	writableDomain := NewWritableDomain()
	writableDomain.Data["vapidPrivateKey"] = "SECRET"

	s := schema.New(DomainSchema())

	require.Nil(t, s.Set(&writableDomain, "themeData.stylesheet", "body { color: red; }"))

	// The theme value lands in themeData...
	require.Equal(t, "body { color: red; }", writableDomain.ThemeData.GetString("stylesheet"))

	// ...and data is left holding only its own secret.
	require.Equal(t, "SECRET", writableDomain.Data["vapidPrivateKey"])
	require.Empty(t, writableDomain.Data["stylesheet"])
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
		readOnlyDomain := NewDomain()
		readOnlyDomain.Hostname = "example.com"

		require.Equal(t, "https://example.com", readOnlyDomain.Host())
		requireAbsolute(t, readOnlyDomain.IconURL())
		requireAbsolute(t, readOnlyDomain.ImageURL())
	})

	t.Run("LocalHostnameUsesHTTP", func(t *testing.T) {
		readOnlyDomain := NewDomain()
		readOnlyDomain.Hostname = "localhost"

		require.Equal(t, "http://localhost", readOnlyDomain.Host())
		requireAbsolute(t, readOnlyDomain.IconURL())
		requireAbsolute(t, readOnlyDomain.ImageURL())
	})

	t.Run("UploadedArtwork", func(t *testing.T) {
		// Uploaded artwork takes the other branch of IconURL/ImageURL
		readOnlyDomain := NewDomain()
		readOnlyDomain.Hostname = "example.com"
		readOnlyDomain.IconID = primitive.NewObjectID()
		readOnlyDomain.ImageID = primitive.NewObjectID()

		requireAbsolute(t, readOnlyDomain.IconURL())
		requireAbsolute(t, readOnlyDomain.ImageURL())
	})

	t.Run("BlankHostnameProducesNoAuthority", func(t *testing.T) {
		// This is the failure the hostname stamp exists to prevent
		readOnlyDomain := NewDomain()

		require.Equal(t, "https://", readOnlyDomain.Host())

		parsed, err := url.Parse(readOnlyDomain.ImageURL())
		require.Nil(t, err)
		require.Empty(t, parsed.Host)

		_, formatErr := urlFormat.Validate(readOnlyDomain.IconURL())
		require.NotNil(t, formatErr)
	})
}

// Every map and slice field starts out allocated, including any field added to Domain later, so a
// record built from the constructor never needs a nil guard before a write.
func TestNewDomain_InitializesEveryMapAndSlice(t *testing.T) {

	value := reflect.ValueOf(NewDomain())

	for index := range value.NumField() {

		field := value.Type().Field(index)

		if kind := field.Type.Kind(); kind != reflect.Map && kind != reflect.Slice {
			continue
		}

		require.False(t, value.Field(index).IsNil(), "NewDomain must initialize %s", field.Name)
	}
}

// newDomainCloneFixture returns a Domain with a value in every map and slice.  Each call returns a
// new record, so one can be cloned and edited while another is kept to compare against.
func newDomainCloneFixture() Domain {

	readOnlyDomain := NewDomain()
	readOnlyDomain.Label = "Original Label"
	readOnlyDomain.Data["sso_secret"] = "original"
	readOnlyDomain.ThemeData["stylesheet"] = "original"
	readOnlyDomain.RegistrationData = mapof.String{"field": "original"}
	readOnlyDomain.Syndication = sliceof.Object[form.LookupCode]{{Value: "bluesky", Label: "Bluesky"}}
	readOnlyDomain.StartupTasks = sliceof.String{"/startup/content"}
	readOnlyDomain.MLSGroupIDs = sliceof.String{"group"}
	readOnlyDomain.Connections["stripe"] = Connection{ProviderID: "stripe", Data: mapof.Any{"mode": "original"}}

	return readOnlyDomain
}

// A clone starts out equal to the Domain it was copied from.
func TestDomain_Clone_CopiesEveryValue(t *testing.T) {

	readOnlyDomain := newDomainCloneFixture()
	require.Equal(t, newDomainCloneFixture(), readOnlyDomain.Clone())
}

// Writing to any map or slice on a clone leaves the original untouched.
func TestDomain_Clone_EditsDoNotReachTheOriginal(t *testing.T) {

	original := newDomainCloneFixture()
	clone := original.Clone()

	clone.Label = "Edited Label"
	clone.Data["sso_secret"] = "edited"
	clone.ThemeData["stylesheet"] = "edited"
	clone.RegistrationData["field"] = "edited"
	clone.Syndication[0].Label = "Edited"
	clone.StartupTasks[0] = "/edited"
	clone.MLSGroupIDs[0] = "edited"
	clone.Connections["stripe"] = Connection{ProviderID: "edited"}
	delete(clone.Data, "sso_secret")

	require.Equal(t, newDomainCloneFixture(), original)
}

// Every map and slice field is copied, including any field added to Domain after Clone was written.
func TestDomain_Clone_CoversEveryMapAndSliceField(t *testing.T) {

	original := newDomainCloneFixture()
	clone := original.Clone()

	originalValue := reflect.ValueOf(original)
	cloneValue := reflect.ValueOf(clone)

	for index := range originalValue.NumField() {

		field := originalValue.Type().Field(index)

		if kind := field.Type.Kind(); kind != reflect.Map && kind != reflect.Slice {
			continue
		}

		// A nil field would pass the pointer check below without proving anything
		require.False(t, originalValue.Field(index).IsNil(), "the fixture must populate %s", field.Name)
		require.NotEqual(t, originalValue.Field(index).Pointer(), cloneValue.Field(index).Pointer(), "Clone must copy %s", field.Name)
	}
}

// A nil map or slice stays nil, so a saved clone writes the same document the original would.
func TestDomain_Clone_KeepsNilFieldsNil(t *testing.T) {

	original := Domain{Label: "Bare"}
	clone := original.Clone()

	require.Equal(t, original, clone)
	require.Nil(t, clone.Data)
	require.Nil(t, clone.ThemeData)
	require.Nil(t, clone.RegistrationData)
	require.Nil(t, clone.Connections)
	require.Nil(t, clone.StartupTasks)
	require.Nil(t, clone.MLSGroupIDs)
	require.Nil(t, clone.Syndication)
}

// An empty map or slice stays empty rather than becoming nil.
func TestDomain_Clone_KeepsEmptyFieldsEmpty(t *testing.T) {

	original := NewDomain()
	clone := original.Clone()

	require.Equal(t, original, clone)
	require.NotNil(t, clone.Data)
	require.NotNil(t, clone.ThemeData)
	require.NotNil(t, clone.RegistrationData)
	require.NotNil(t, clone.Connections)
	require.NotNil(t, clone.StartupTasks)
	require.NotNil(t, clone.MLSGroupIDs)
	require.NotNil(t, clone.Syndication)
}

// Values nested inside a copied map are still shared, as the Clone header states.
func TestDomain_Clone_SharesNestedValues(t *testing.T) {

	original := newDomainCloneFixture()
	clone := original.Clone()

	clone.Connections["stripe"].Data["mode"] = "edited"

	require.Equal(t, "edited", original.Connections["stripe"].Data["mode"])
}
