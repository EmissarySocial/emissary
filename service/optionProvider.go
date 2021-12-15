package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
)

type OptionProvider struct {
	User *User
}

func NewOptionProvider(user *User) OptionProvider {
	return OptionProvider{
		User: user,
	}
}

func (service OptionProvider) OptionCodes(path string) ([]form.OptionCode, error) {

	path = list.Last(path, "/")

	switch path {
	case "users":
		it, err := service.User.List(nil)

		if err != nil {
			return nil, derp.Wrap(err, "ghost.service.OptionProvider.OptionCodes", "Error connecting to database")
		}

		record := model.NewUser()
		result := make([]form.OptionCode, it.Count())
		for it.Next(&record) {
			result = append(result, form.OptionCode{
				Label: record.DisplayName,
				Value: record.UserID.Hex(),
			})
			record = model.NewUser()
		}

		return result, nil
	}

	return nil, derp.New(500, "ghost.service.OptionProvider.OptionCodes", "Unrecognized Path: ", path)
}
