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

	// statusMessage keeps its 8192-character limit, though the Template redeclares it
	statusMessage, isString := merged.Properties["statusMessage"].(schema.String)
	require.True(t, isString)
	require.Equal(t, 8192, statusMessage.MaxLength)

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

// newNormalizableUser returns a User whose required fields are filled, so Normalize reaches its data
func newNormalizableUser() User {
	user := NewUser()
	user.DisplayName = "Sarah Connor"
	user.EmailAddress = "sarah@connor.example"
	user.Username = "sarah"
	return user
}

// TestUserSchemaForTemplate_NormalizeKeepsADeclaredFormat confirms that a format the Template declares
// survives Normalize, where UserSchema alone applies its no-html default (User.Save normalizes with both)
func TestUserSchemaForTemplate_NormalizeKeepsADeclaredFormat(t *testing.T) {

	templateSchema := schema.New(schema.Object{
		Properties: schema.ElementMap{
			"data": schema.Object{
				Properties: schema.ElementMap{
					"notes": schema.String{Format: "markdown"},
				},
			},
		},
	})

	notes := "## Shows\n\nFriday > Saturday\n\nSee you there"

	// UserSchema alone strips the text down to one line
	user := newNormalizableUser()
	user.Data["notes"] = notes
	_, err := schema.New(UserSchema()).Normalize(&user)
	require.NoError(t, err)
	require.NotEqual(t, notes, user.Data["notes"])

	// The Template's own format keeps it as written
	user = newNormalizableUser()
	user.Data["notes"] = notes
	_, err = schema.New(UserSchemaForTemplate(templateSchema)).Normalize(&user)
	require.NoError(t, err)
	require.Equal(t, notes, user.Data["notes"])
}

// TestUserSchemaForTemplate_NormalizeRefusesABrokenStoredValue confirms that a stored value its Template's
// format refuses makes Normalize fail, so User.Save refuses the whole record until the value is repaired
func TestUserSchemaForTemplate_NormalizeRefusesABrokenStoredValue(t *testing.T) {

	user := newNormalizableUser()
	user.Data["color-body"] = "#abc"

	_, err := schema.New(UserSchemaForTemplate(bandwagonLikeSchema())).Normalize(&user)
	require.Error(t, err)

	// UserSchema alone accepts any string, which is how such a value was stored
	_, err = schema.New(UserSchema()).Normalize(&user)
	require.NoError(t, err)
}
