package render

import (
	"bytes"
	"html/template"
	"math/rand"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content/transaction"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// UpdateDraft manages the content.Content in a stream.
type UpdateDraft struct {
	factory Factory
	model.ActionConfig
}

func NewAction_UpdateDraft(factory Factory, config model.ActionConfig) UpdateDraft {
	return UpdateDraft{
		factory:      factory,
		ActionConfig: config,
	}
}

func (action UpdateDraft) Get(renderer Renderer) (string, error) {

	var result bytes.Buffer

	// Try to load the draft from the database, overwriting the stream already in the renderer
	service := action.factory.StreamDraft()

	if err := service.LoadByID(renderer.stream.StreamID, &renderer.stream); err != nil {
		return "", derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	t := action.template()

	if err := t.Execute(&result, renderer); err != nil {
		return "", derp.Wrap(err, "ghost.render.UpdateDraft.Get", "Error executing template")
	}

	return result.String(), nil
}

func (action UpdateDraft) Post(ctx *steranko.Context, stream *model.Stream) error {

	var draft model.Stream

	// Try to load the stream draft from the database
	service := action.factory.StreamDraft()

	if err := service.LoadByID(stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	// Try to parse the body content into a transaction
	body := make(map[string]interface{})

	if err := ctx.Bind(&body); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error binding data"))
	}

	txn, err := transaction.Parse(body)

	if err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error parsing transaction", body))
	}

	// Try to execute the transaction
	if err := txn.Execute(&(draft.Content)); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error executing transaction", txn))
	}

	// Try to save the draft

	if err := service.Save(&draft, "edit content: "+txn.Description()); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error saving stream"))
	}

	// Return response to caller
	return ctx.String(200, convert.String(rand.Int63()))

	// ctx.Response().Header().Add("HX-Redirect", "/"+stream.Token)
	// return ctx.NoContent(200)

}

// template retrieves the templpate paramer from the ActionConfig.
// IF this parameter is missing for some reason, it returns an empty template
func (action UpdateDraft) template() *template.Template {

	if t := action.GetInterface("template"); t != nil {

		if result, ok := t.(*template.Template); ok {
			return result
		}
	}

	return template.New("missing")
}
