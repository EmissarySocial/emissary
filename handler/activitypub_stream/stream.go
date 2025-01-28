package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

func GetJSONLD(ctx *steranko.Context, factory *domain.Factory, stream *model.Stream) error {

	const location = "activitypub_stream.GetJSONLD"

	// Load the Template
	templateService := factory.Template()
	streamService := factory.Stream()

	template, err := templateService.Load(stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading Template", stream.TemplateID)
	}

	// If this Stream is not an Actor, then just return a standard JSON-LD response.
	if template.Actor.IsNil() {
		jsonld := streamService.JSONLD(stream)
		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		return ctx.JSON(http.StatusOK, jsonld)
	}

	// Try to locate the domain
	// Try to load the Encryption Key for this Actor
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()
	if err := keyService.LoadByParentID(model.EncryptionKeyTypeStream, stream.StreamID, &key); err != nil {
		return derp.Wrap(err, location, "Error loading Public Key", stream.StreamID)
	}

	// Combine the Actor and the Public Key
	result := template.Actor.JSONLD(stream)
	result[vocab.PropertyPublicKey] = mapof.Any{
		vocab.PropertyID:   stream.Permalink() + "#main-key",
		vocab.PropertyType: "Key",
		"owner":            stream.Permalink(),
		"publicKeyPem":     key.PublicPEM,
	}

	// Return an ActivityPub response
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(http.StatusOK, result)
}
