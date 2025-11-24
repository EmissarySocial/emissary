package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetUserExportCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetUserExportCollection"

	requestURL := fullURL(factory, ctx)

	// Locate the service to use
	exportService := factory.Export()
	collection := ctx.Param("collection")
	service, err := exportService.FindService(collection)

	// If we can't locate the service, then just return a 404 with an empty collection
	if err != nil {
		ctx.Response().Header().Set("Content-Type", "application/activity+json")
		return ctx.JSON(http.StatusNotFound, streams.NewOrderedCollection(requestURL))
	}

	// Generate the export collection for this service
	records, err := service.ExportCollection(session, user.UserID)

	if err != nil {
		return derp.Wrap(err, location, "Uable to retrieve exportable records", collection)
	}

	// Return the result to the caller as a JSON-LD Collection
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	result := streams.NewOrderedCollection(requestURL)
	result.TotalItems = len(records)
	result.OrderedItems = slice.Map(records, func(recordID model.IDOnly) any {
		return requestURL + "/" + recordID.ID.Hex()
	})

	return ctx.JSON(http.StatusOK, result)
}

func GetUserExportDocument(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.GetUserExportDocument"

	// Locate the service to use
	exportService := factory.Export()
	collection := ctx.Param("collection")
	service, err := exportService.FindService(collection)

	if err != nil {
		return derp.Wrap(err, location, "Unable to find export service", collection)
	}

	recordID, err := primitive.ObjectIDFromHex(ctx.Param("recordId"))

	if err != nil {
		return derp.Wrap(err, location, "Invalid RecordID", ctx.Param("recordId"))
	}

	// Generate the export collection for this service
	record, err := service.ExportDocument(session, user.UserID, recordID)

	if err != nil {
		return derp.Wrap(err, location, "Uable to retrieve exportable record", collection, recordID)
	}

	// Return the result to the caller as a JSON-LD Collection
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.String(http.StatusOK, record)

}
