package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mastodon API
 ******************************************/

func (service *Stream) QueryByUser(session data.Session, userID primitive.ObjectID, criteria exp.Expression, options ...option.Option) ([]model.Stream, error) {

	criteria = criteria.AndEqual("ownerId", userID)
	options = append(options, option.SortDesc("createDate"))

	return service.Query(session, criteria, options...)
}
