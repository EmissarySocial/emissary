package build

import (
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/form"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/mapof"
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

// TestEditModelObject_OptionTemplates renders one cached step for two Streams, and requires
// that each form carries its own Stream's ID while the cached form keeps its templates
func TestEditModelObject_OptionTemplates(t *testing.T) {

	// BUG-204: rendering wrote each option back into the cached form, so the first request
	// after a start or reload fixed every later form to its ID.
	widget.UseAll()
	cached := newOptionTemplateStep(t)

	first, second := newOptionTemplateStream(), newOptionTemplateStream()
	firstHTML := renderOptionTemplateStep(t, cached, &first)
	secondHTML := renderOptionTemplateStep(t, cached, &second)

	require.Contains(t, firstHTML, "/.validate/stream/token?streamId="+first.ID())
	require.Contains(t, firstHTML, `hx-post="/`+first.ID()+`/delete-icon"`)
	require.Contains(t, secondHTML, "/.validate/stream/token?streamId="+second.ID())
	require.Contains(t, secondHTML, `hx-post="/`+second.ID()+`/delete-icon"`)
	require.NotContains(t, secondHTML, first.ID())

	// The cached form still holds every template
	require.Equal(t, "/.validate/stream/token?streamId={{.ID}}", cached.Form.Children[0].Children[0].Options.GetString("validator", nil))
	require.Equal(t, "/{{.ID}}/delete-icon", cached.Form.Children[1].Options.GetString("delete", nil))
}

// TestEditModelObject_OptionTemplates_Concurrent renders one cached step from many goroutines,
// which the race detector reports if any render writes the shared form
func TestEditModelObject_OptionTemplates_Concurrent(t *testing.T) {

	widget.UseAll()
	cached := newOptionTemplateStep(t)
	mismatches := make(chan string, 20)

	var wait sync.WaitGroup
	for range 20 {
		wait.Go(func() {
			stream := newOptionTemplateStream()
			result, err := form.Editor(schema.New(model.StreamSchema()), cached.getForm(optionTemplateBuilder{}), &stream, nil)

			// Report a mismatch rather than failing here, because require cannot run off the test goroutine
			if (err != nil) || !strings.Contains(result, "streamId="+stream.ID()) {
				mismatches <- stream.ID()
			}
		})
	}

	wait.Wait()
	close(mismatches)

	for streamID := range mismatches {
		t.Errorf("render for %q did not carry its own ID", streamID)
	}
}

// TestEditModelObject_PropertyForm requires that a step without a form uses the Domain's
// PropertyForm, and that a step with one ignores it
func TestEditModelObject_PropertyForm(t *testing.T) {

	themeForm := form.Element{Type: "layout-vertical", Children: []form.Element{{Type: "text", Path: "label"}}}
	builder := optionTemplatePropertyBuilder{propertyForm: themeForm}

	require.Equal(t, themeForm, StepEditModelObject{}.getForm(builder))

	cached := newOptionTemplateStep(t)
	require.Equal(t, cached.Form, cached.getForm(builder))
}

// TestEditModelObject_EmptyForm requires that a step with no form, and no PropertyForm to fall
// back on, returns its empty form
func TestEditModelObject_EmptyForm(t *testing.T) {
	require.True(t, StepEditModelObject{}.getForm(optionTemplateBuilder{}).IsEmpty())
}

// newOptionTemplateStep parses an edit step shaped like the Stream templates that use option
// templates, with one option one level down and one two levels down
func newOptionTemplateStep(t *testing.T) StepEditModelObject {

	t.Helper()

	parsed, err := step.NewEditModelObject(mapof.Any{
		"form": map[string]any{
			"type": "layout-tabs",
			"children": []any{
				map[string]any{
					"type": "layout-vertical",
					"children": []any{
						map[string]any{"type": "text", "path": "token", "options": map[string]any{"autocomplete": "off", "validator": "/.validate/stream/token?streamId={{.ID}}"}},
					},
				},
				map[string]any{"type": "upload", "path": "iconUrl", "options": map[string]any{"accept": "image/*", "delete": "/{{.ID}}/delete-icon"}},
			},
		},
	})

	require.NoError(t, err)
	return StepEditModelObject(parsed)
}

// newOptionTemplateStream returns a Stream with an icon, so that the upload widget draws its
// delete button
func newOptionTemplateStream() model.Stream {
	stream := model.NewStream()
	stream.IconURL = "https://example.com/icon.webp"
	return stream
}

// renderOptionTemplateStep draws the step's form for one Stream, as StepEditModelObject.Get does
func renderOptionTemplateStep(t *testing.T, cached StepEditModelObject, stream *model.Stream) string {

	t.Helper()

	result, err := form.Editor(schema.New(model.StreamSchema()), cached.getForm(optionTemplateBuilder{}), stream, nil)
	require.NoError(t, err)
	return result
}

// optionTemplateBuilder is a Builder with no PropertyForm
type optionTemplateBuilder struct {
	Builder
}

// optionTemplatePropertyBuilder is a Builder that supplies a Theme's property form
type optionTemplatePropertyBuilder struct {
	Builder
	propertyForm form.Element
}

// PropertyForm returns the Theme's form, as the Domain builder does
func (builder optionTemplatePropertyBuilder) PropertyForm() form.Element {
	return builder.propertyForm
}
