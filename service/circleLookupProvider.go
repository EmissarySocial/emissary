package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CircleLookupProvider struct {
	circleService *Circle
	userID        primitive.ObjectID
}

func NewCircleLookupProvider(circleService *Circle, userID primitive.ObjectID) CircleLookupProvider {
	return CircleLookupProvider{
		circleService: circleService,
	}
}

func (service CircleLookupProvider) Get() []form.LookupCode {
	circles, _ := service.circleService.QueryByUser(service.userID)

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

	if err := service.circleService.Save(&circle, "created"); err != nil {
		return "", derp.Wrap(err, "service.CircleLookupProvider.Add", "Error saving circle", name)
	}

	return circle.ID(), nil
}
