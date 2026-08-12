// Package activitypub_stream serves the ActivityPub representation of a Stream.
//
// A Stream is an actor only when its Template defines one; otherwise it is served as a plain
// JSON-LD object.  Both shapes come out of the same endpoint, negotiated by the caller.
package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
)

// GetJSONLD serves the ActivityPub representation of a Stream, as either an Actor or a plain object
func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.GetJSONLD"

	// RULE: The permissions in the HTTP signature must satisfy the Stream's required permissions
	permissionService := factory.Permission()
	permissions := permissionService.ParseHTTPSignature(session, ctx.Request()) // nolint:scopeguard

	if !slice.ContainsAny(stream.DefaultAllow, permissions...) {
		return derp.Forbidden(location, "You do not have permission to view this content")
	}

	// If this Stream is not an Actor, then just return a standard JSON-LD response
	if template.Actor.IsNil() {
		jsonld := factory.Stream().JSONLD(session, stream)
		headers.SetAll(ctx.Response().Header(), headers.VariantActivityPub, stream)
		return ctx.JSON(http.StatusOK, jsonld)
	}

	// Load the Encryption Key that this Actor signs with
	keyService := factory.EncryptionKey()
	key := model.NewEncryptionKey()
	if err := keyService.LoadByParentID(session, model.EncryptionKeyTypeStream, stream.StreamID, &key); err != nil {
		return derp.Wrap(err, location, "Loading Public Key", stream.StreamID)
	}

	// Combine the Actor and the Public Key.  Key ID/owner must use the actor `id` -- it is what
	// this Actor signs with, and what remote servers dereference when verifying our signatures.
	actorID := stream.ActivityPubURL()

	result := template.Actor.JSONLD(stream)
	result[vocab.PropertyPublicKey] = mapof.Any{
		vocab.PropertyID:           actorID + "#main-key",
		vocab.PropertyOwner:        actorID,
		vocab.PropertyPublicKeyPEM: key.PublicPEM,
	}

	// Return an ActivityPub response
	headers.SetAll(ctx.Response().Header(), headers.VariantActivityPub, stream)
	return ctx.JSON(http.StatusOK, result)
}
