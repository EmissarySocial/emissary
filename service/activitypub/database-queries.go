package activitypub

import (
	"net/url"

	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

func (db *Database) queryAllURLs(modelType string, actorURL *url.URL, criteria exp.Expression) ([]string, error) {

	const location = "service.activitypub.queryAllIRIsByURL"

	type hasAllURLs interface {
		QueryAllURLS(exp.Expression) ([]string, error)
	}

	// Parse the URL into a UserID
	userID, _, _, err := parseURL(actorURL)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error parsing URL", actorURL)
	}

	// Build the query criteria
	criteria = criteria.
		AndEqual("userId", userID).
		AndEqual("journal.deleteDate", 0)

	// Get the corresponding Model Service
	modelService, err := db.factory.Model(modelType)

	if err != nil {
		return nil, derp.Wrap(err, location, "Error getting model service", modelType)
	}

	// Confirm that the model service has a "QueryAllURLs" method
	if queryable, ok := modelService.(hasAllURLs); ok {
		return queryable.QueryAllURLS(criteria)
	}

	return nil, derp.NewInternalError(location, "ModelService does not implement QueryAllURLS() method")
}
