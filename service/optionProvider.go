package service

import (
	"github.com/benpate/convert"
	"github.com/benpate/data"
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

func (service OptionProvider) OptionCodes(path string) []form.OptionCode {

	path = list.Last(path, "/")

	switch path {
	case "users":
		if it, err := service.User.List(nil); err == nil {
			return mapOptionCodes(it, "_id", "displayName")
		}
	}

	return []form.OptionCode{}
}

func mapOptionCodes(it data.Iterator, values string, labels string) []form.OptionCode {

	result := make([]form.OptionCode, 0)

	if it == nil {
		return result
	}

	data := model.NewGeneric()

	for it.Next(data) {
		result = append(result, form.OptionCode{
			Label: convert.String(data[labels]),
			Value: convert.String(data[values]),
		})
		data = model.NewGeneric()
	}

	return result
}
