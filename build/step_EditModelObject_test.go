package build

import (
	"net/url"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestEditModelObject_Multiselect verifies that a `multiselect` widget bound to an
// Array-typed schema path actually writes back into the model object.  This mirrors the
// "Groups" tab of the admin-users edit form, which silently dropped every selection when
// id.Slice could not decode the *sliceof.String value that multi-value widgets post.
func TestEditModelObject_Multiselect(t *testing.T) {

	widget.UseAll()

	// The Groups tab of _embed/templates/admin-users/template.hjson
	element := form.Element{
		Type: "layout-tabs",
		Children: []form.Element{
			{
				Type:  "layout-vertical",
				Label: "Groups",
				Children: []form.Element{
					{Type: "multiselect", Path: "groupIds", Options: map[string]any{"provider": "groups", "sort": false}},
				},
			},
		},
	}

	editForm := form.New(schema.New(model.UserSchema()), element)

	{ // Checking two groups adds both to the User
		user := model.NewUser()
		values := url.Values{"groupIds": []string{"000000000000000000000002", "000000000000000000000003"}}

		require.NoError(t, editForm.SetURLValues(&user, values, nil))
		require.Equal(t, []string{"000000000000000000000002", "000000000000000000000003"}, user.GroupIDs.SliceOfString())
	}

	{ // Un-checking every group clears the User's existing groups
		user := model.NewUser()
		require.NoError(t, editForm.SetURLValues(&user, url.Values{"groupIds": []string{"000000000000000000000002"}}, nil))
		require.Equal(t, 1, user.GroupIDs.Length())

		require.NoError(t, editForm.SetURLValues(&user, url.Values{}, nil))
		require.Zero(t, user.GroupIDs.Length())
	}
}

// TestEditModelObject_Multiselect_Circle covers the second id.Slice-backed multiselect in
// the templates: the "Products" tab of the user-settings circle editor.
func TestEditModelObject_Multiselect_Circle(t *testing.T) {

	widget.UseAll()

	element := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "multiselect", Path: "productIds", Options: map[string]any{"provider": "merchantAccounts-all-products"}},
		},
	}

	editForm := form.New(schema.New(model.CircleSchema()), element)
	circle := model.NewCircle()
	values := url.Values{"productIds": []string{"000000000000000000000002"}}

	require.NoError(t, editForm.SetURLValues(&circle, values, nil))
	require.Equal(t, []string{"000000000000000000000002"}, circle.ProductIDs.SliceOfString())
}

// TestEditModelObject_Multiselect_SliceOfString guards the other half of the multiselect
// story: paths backed by sliceof.String rather than id.Slice.
func TestEditModelObject_Multiselect_SliceOfString(t *testing.T) {

	widget.UseAll()

	element := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "multiselect", Path: "events", Options: map[string]any{"provider": "webhook-types"}},
		},
	}

	editForm := form.New(schema.New(model.WebhookSchema()), element)
	webhook := model.NewWebhook()
	values := url.Values{"events": []string{model.WebhookEventUserCreate}}

	require.NoError(t, editForm.SetURLValues(&webhook, values, nil))
	require.Equal(t, []string{model.WebhookEventUserCreate}, []string(webhook.Events))
}

// TestEditModelObject_Multiselect_DeltaSlice covers the third and last slice type behind a
// multiselect: delta.Slice, which backs Stream.Syndication and the "Distribute to Streaming
// Stations" control in the bandwagon-album template.  delta.Slice was missed when id.Slice
// was taught to accept the *sliceof.String that multi-value widgets post, so every
// syndication selection was silently discarded.
func TestEditModelObject_Multiselect_DeltaSlice(t *testing.T) {

	widget.UseAll()

	element := form.Element{
		Type: "layout-vertical",
		Children: []form.Element{
			{Type: "multiselect", Path: "syndication", Options: map[string]any{"provider": "syndication-targets"}},
		},
	}

	editForm := form.New(schema.New(model.StreamSchema()), element)

	{ // Checking two targets adds both to the Stream
		stream := model.NewStream()
		values := url.Values{"syndication": []string{"bandwagon", "spotify"}}

		require.NoError(t, editForm.SetURLValues(&stream, values, nil))
		require.Equal(t, []string{"bandwagon", "spotify"}, stream.Syndication.Values)
		require.Equal(t, []string{"bandwagon", "spotify"}, stream.Syndication.Added)
		require.Empty(t, stream.Syndication.Deleted)
	}

	{ // Un-checking every target clears the Stream, and records the removals
		stream := model.NewStream()
		require.NoError(t, editForm.SetURLValues(&stream, url.Values{"syndication": []string{"bandwagon"}}, nil))
		stream.Syndication.Reset()

		require.NoError(t, editForm.SetURLValues(&stream, url.Values{}, nil))
		require.Empty(t, stream.Syndication.Values)
		require.Equal(t, []string{"bandwagon"}, stream.Syndication.Deleted)
	}

	// RULE: Re-saving a Stream with the SAME targets ticked is not a change.  Stream.publish
	// broadcasts Syndication.Deleted to every partner, so a phantom diff here would send a
	// "delete" to services the album is still supposed to be on.
	{
		stream := model.NewStream()
		require.NoError(t, editForm.SetURLValues(&stream, url.Values{"syndication": []string{"bandwagon", "spotify"}}, nil))
		stream.Syndication.Reset()

		values := url.Values{"syndication": []string{"bandwagon", "spotify"}}
		require.NoError(t, editForm.SetURLValues(&stream, values, nil))

		require.Equal(t, []string{"bandwagon", "spotify"}, stream.Syndication.Values)
		require.Empty(t, stream.Syndication.Added)
		require.Empty(t, stream.Syndication.Deleted)
		require.False(t, stream.Syndication.IsChanged())
	}
}
