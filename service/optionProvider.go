package service

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/list"
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

	case "sharing":
		return []form.OptionCode{
			{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			{Value: "authenticated", Label: "Authenticated People Only"},
			{Value: "private", Label: "Only Selected Groups"},
		}, nil

	case "groups":
		return service.Group.ListAsOptions()
	}

	return nil, derp.New(500, "service.OptionProvider.OptionCodes", "Unrecognized Path: ", path)
}
