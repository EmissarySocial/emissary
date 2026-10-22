package model

import (
	"strings"
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// bandwagonLikeSchema returns an outbox Template schema shaped like bandwagon-outbox's
func bandwagonLikeSchema() schema.Schema {
	return schema.New(schema.Object{
		Properties: schema.ElementMap{
			"statusMessage": schema.String{},
			"data": schema.Object{
				Wildcard: schema.String{},
				Properties: schema.ElementMap{
					"color-body":          schema.String{Format: "color"},
					"musicbrainzArtistId": schema.String{Format: "uuid"},
					"label-albums":        schema.String{MaxLength: 10000},
					"layout-highlight":    schema.Boolean{},
				},
			},
		},
	})
}

// TestUserSchemaForTemplate_EnforcesDataFormats confirms that a Template's data formats reject bad values
func TestUserSchemaForTemplate_EnforcesDataFormats(t *testing.T) {

	s := schema.New(UserSchemaForTemplate(bandwagonLikeSchema()))
	user := NewUser()

	require.NoError(t, s.Set(&user, "data.color-body", "#336699"))
	require.Equal(t, "#336699", user.Data["color-body"])
	require.Error(t, s.Set(&user, "data.color-body", "not-a-color"))

	require.NoError(t, s.Set(&user, "data.musicbrainzArtistId", "5b11f4ce-a62d-471e-81fc-a69a8278c7da"))
	require.Error(t, s.Set(&user, "data.musicbrainzArtistId", "not-a-uuid"))
}

// TestUserSchemaForTemplate_KeepsTheDataLimit confirms that no data key may exceed 4096 characters
func TestUserSchemaForTemplate_KeepsTheDataLimit(t *testing.T) {

	s := schema.New(UserSchemaForTemplate(bandwagonLikeSchema()))
	long := strings.Repeat("x", 5000)

	// A declared key with a higher limit, and an undeclared key under the wildcard
	for _, path := range []string{"data.label-albums", "data.anything-else"} {
		user := NewUser()
		_ = s.Set(&user, path, long)
		require.LessOrEqual(t, len(user.Data[strings.TrimPrefix(path, "data.")]), userDataMaxLength, path)
	}
}

// TestUserSchemaForTemplate_KeepsUserSchemaRules confirms that a Template cannot loosen the stock fields
func TestUserSchemaForTemplate_KeepsUserSchemaRules(t *testing.T) {

	merged, isObject := UserSchemaForTemplate(bandwagonLikeSchema()).(schema.Object)
	require.True(t, isObject)

	// statusMessage keeps its 2048-character limit, though the Template redeclares it
	statusMessage, isString := merged.Properties["statusMessage"].(schema.String)
	require.True(t, isString)
	require.Equal(t, 2048, statusMessage.MaxLength)

	// A non-string data rule is ignored, because data holds only strings
	data, isObject := merged.Properties["data"].(schema.Object)
	require.True(t, isObject)
	require.NotContains(t, data.Properties, "layout-highlight")
	require.Equal(t, schema.String{MaxLength: userDataMaxLength}, data.Wildcard)
}

// TestUserSchemaForTemplate_NoTemplateSchema confirms that an empty Template schema yields UserSchema unchanged
func TestUserSchemaForTemplate_NoTemplateSchema(t *testing.T) {
	require.Equal(t, UserSchema(), UserSchemaForTemplate(schema.Schema{}))
	require.Equal(t, UserSchema(), UserSchemaForTemplate(schema.New(schema.Object{Properties: schema.ElementMap{}})))
}

// TestUserSchemaForTemplate_MusicBrainzURL confirms that a pasted MusicBrainz page URL is stored as its artist ID
func TestUserSchemaForTemplate_MusicBrainzURL(t *testing.T) {

	s := schema.New(UserSchemaForTemplate(bandwagonLikeSchema()))
	user := NewUser()

	require.NoError(t, s.Set(&user, "data.musicbrainzArtistId", "https://musicbrainz.org/artist/5B11F4CE-A62D-471E-81FC-A69A8278C7DA/"))
	require.Equal(t, "5b11f4ce-a62d-471e-81fc-a69a8278c7da", user.Data["musicbrainzArtistId"])
}
