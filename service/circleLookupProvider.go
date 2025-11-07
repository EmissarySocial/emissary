package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CircleLookupProvider struct {
	circleService *Circle
	userID        primitive.ObjectID
	session       data.Session
}

func NewCircleLookupProvider(session data.Session, circleService *Circle, userID primitive.ObjectID) CircleLookupProvider {
	return CircleLookupProvider{
		circleService: circleService,
		userID:        userID,
		session:       session,
	}
}

func (service CircleLookupProvider) Get() []form.LookupCode {
	circles, err := service.circleService.QueryByUser(service.session, service.userID, option.SortAsc("name"))

	if err != nil {
		derp.Report(derp.Wrap(err, "service.CircleLookupProvider.Get", "Error retrieving circles for user", service.userID.Hex()))
	}

	result := make([]form.LookupCode, 0, len(circles))

	for _, circle := range circles {
		result = append(result, circle.LookupCode())
	}

	return result
}

func (service CircleLookupProvider) Add(name string) (string, error) {

	circle := model.NewCircle()
	circle.Name = name
	circle.UserID = service.userID

	if err := service.circleService.Save(service.session, &circle, "created"); err != nil {
		return "", derp.Wrap(err, "service.CircleLookupProvider.Add", "Unable to save circle", name)
	}

	return circle.ID(), nil
}
