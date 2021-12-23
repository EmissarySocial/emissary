package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/list"
)

type OptionProvider struct {
	Group *Group
	User  *User
}

func NewOptionProvider(group *Group, user *User) OptionProvider {
	return OptionProvider{
		Group: group,
		User:  user,
	}
}

func (service OptionProvider) OptionCodes(path string) ([]form.OptionCode, error) {

	path = list.Last(path, "/")

	switch path {

	case "groups":
		return service.Group.ListAsOptions()
	}

	return nil, derp.New(500, "ghost.service.OptionProvider.OptionCodes", "Unrecognized Path: ", path)
}
