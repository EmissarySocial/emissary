package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *NewsFeed) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *NewsFeed) ExportDocument(session data.Session, userID primitive.ObjectID, newsItemID primitive.ObjectID) (string, error) {

	const location = "service.NewsFeed.ExportDocument"

	// Load the NewsFeed
	newsItem := model.NewNewsItem()
	if err := service.LoadByID(session, userID, newsItemID, &newsItem); err != nil {
		return "", derp.Wrap(err, location, "Unable to load NewsFeed")
	}

	// Marshal the newsItem as JSON
	result, err := json.Marshal(newsItem)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal NewsFeed", newsItem)
	}

	// Success
	return string(result), nil
}
