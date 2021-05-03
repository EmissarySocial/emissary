package handler

import (
	"math/rand"

	"github.com/benpate/choose"
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

///////////////////////////////////////////////////////
// REQUEST HANDLERS

// GetStreamDraft returns an echo.HandlerFunc that displays a transition form
func GetStreamDraft(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to load the draft
		factory, draft, err := loadStreamDraft(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStreamDraft", "Domain Error"))
		}

		// Get the renderer
		renderer := factory.StreamEditor(ctx, draft)

		// Render the draft stream
		return derp.Report(renderStream(ctx, factory, renderer))
	}
}

// PostStreamDraft updates a draft value for a stream
func PostStreamDraft(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to load the draft
		factory, draft, err := loadStreamDraft(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error Loading Stream"))
		}

		// Try to parse the body content into a transaction
		body := make(map[string]interface{})

		if err := ctx.Bind(&body); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error binding data"))
		}

		transaction, err := content.ParseTransaction(body)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error parsing transaction", body))
		}

		// Try to execute the transaction
		if err := transaction.Execute(&(draft.Content)); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error executing transaction", transaction))
		}

		// Try to save the draft
		service := factory.StreamDraft()

		if err := service.Save(draft, "edit content: "+transaction.Description()); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error saving stream"))
		}

		// Return response to caller
		ctx.String(200, convert.String(rand.Int63()))
		// ctx.Response().Header().Add("HX-Redirect", "/"+stream.Token)
		return ctx.NoContent(200)
	}
}

// PublishStreamDraft updates a stream with the data in the draft
func PublishStreamDraft(factoryManager *server.FactoryManager) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		// Try to load the draft
		factory, draft, err := loadStreamDraft(factoryManager, ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error Loading Stream"))
		}

		// Try to save the draft into the Stream collection
		service := factory.Stream()

		if err := service.Save(draft, ""); err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error publishing draft"))
		}

		// Try to delete the draft.
		// It's ok to fail silently because we have already published this to the main collection
		draftService := factory.StreamDraft()

		if err := draftService.Delete(draft, "published"); err != nil {
			derp.Report(derp.Wrap(err, "ghost.handler.PublishStreamDraft", "Error deleting published draft"))
		}

		ctx.Response().Header().Add("HX-Redirect", "/"+draft.Token)
		return ctx.NoContent(200)
	}
}

///////////////////////////////////////////////////////////
// UTILITIES

// loadStreamDraft loads an existing draft from the domain hierarchy
func loadStreamDraft(factoryManager *server.FactoryManager, ctx echo.Context) (*domain.Factory, *model.Stream, error) {

	// Get the service factory
	factory, err := factoryManager.ByContext(ctx)

	if err != nil {
		return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.loadStream", "Unrecognized domain"))
	}

	// Get the stream service
	service := factory.StreamDraft()

	// Get the stream
	token := choose.String(ctx.Param("stream"), "home")
	draft, err := service.LoadByToken(token)

	if err != nil {
		if !derp.NotFound(err) {
			return nil, nil, derp.Report(derp.Wrap(err, "ghost.handler.loadStream", "Error loading stream"))
		}
	}

	return factory, draft, nil
}
