package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
)

func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetJSONLD"

	// Verify permissions by checking the required permissions (stream.DefaultAllow) against the permissions in the request signature
	permissionService := factory.Permission()
	permissions := permissionService.ParseHTTPSignature(session, ctx.Request())

	if !slice.ContainsAny(stream.DefaultAllow, permissions...) {
		return derp.ForbiddenError(location, "You do not have permission to view this content")
	}

	streamService := factory.Stream()

	// If this Stream is not an Actor, then just return a standard JSON-LD response.
	if template.Actor.IsNil() {
		jsonld := streamService.JSONLD(session, stream)
		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		return ctx.JSON(http.StatusOK, jsonld)
	}

	// Try to locate the domain
	// Try to load the Encryption Key for this Actor
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()
	if err := keyService.LoadByParentID(session, model.EncryptionKeyTypeStream, stream.StreamID, &key); err != nil {
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
