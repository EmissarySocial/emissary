package mastodon

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestSortedProfileFields orders fields by their numeric position, so "10" comes after "2"
func TestSortedProfileFields(t *testing.T) {

	fields := map[string]profileField{
		"10": {Name: "Ten"},
		"2":  {Name: "Two"},
		"0":  {Name: "Zero"},
	}

	result := sortedProfileFields(fields)
	require.Equal(t, []string{"Zero", "Two", "Ten"}, []string{result[0].Name, result[1].Name, result[2].Name})
}

// TestProfileLinksFromFields covers blank rows, bare addresses, kept link data, and refused values
func TestProfileLinksFromFields(t *testing.T) {

	userID := primitive.NewObjectID()
	existing := []model.PersonLink{{Name: "Old label", ProfileURL: "https://example.com/me", UserID: userID}}

	links, err := profileLinksFromFields(existing, []profileField{
		{Name: "Website", Value: " https://example.com/me "},
		{Name: "", Value: ""},
		{Name: "Code", Value: "git.example.com/demo"},
	})

	require.NoError(t, err)
	require.Len(t, links, 2, "a blank row is dropped")
	require.Equal(t, "Website", links[0].Name, "the label is updated")
	require.Equal(t, userID, links[0].UserID, "a link the User already had keeps its other data")
	require.Equal(t, "https://git.example.com/demo", links[1].ProfileURL, "a bare address gets https://")

	// Clearing every field leaves no links
	links, err = profileLinksFromFields(existing, nil)
	require.NoError(t, err)
	require.Empty(t, links)

	// Values Emissary cannot keep as a link are refused, not silently dropped
	for _, field := range []profileField{
		{Name: "Pronouns", Value: "they/them"},
		{Name: "Script", Value: "javascript:alert(1)"},
		{Name: "", Value: "https://example.com"},
	} {
		_, err := profileLinksFromFields(existing, []profileField{field})
		require.Error(t, err, field)
		require.Equal(t, http.StatusUnprocessableEntity, derp.ErrorCode(err), field)
	}
}

// TestUser_TootSourceFields confirms the editable fields carry the plain address, not link markup
func TestUser_TootSourceFields(t *testing.T) {

	user := model.NewUser()
	user.Links = append(user.Links,
		model.PersonLink{Name: "Website", ProfileURL: "https://example.com"},
		model.PersonLink{Name: "", ProfileURL: "https://skipped.example"},
	)

	fields := user.TootSourceFields()
	require.Len(t, fields, 1)
	require.Equal(t, "Website", fields[0].Name)
	require.Equal(t, "https://example.com", fields[0].Value)
}
