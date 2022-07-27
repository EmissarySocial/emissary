package service

import (
	"github.com/benpate/form"
)

type LookupProvider struct {
	Group *Group
	User  *User
}

func NewLookupProvider(group *Group, user *User) LookupProvider {
	return LookupProvider{
		Group: group,
		User:  user,
	}
}

func (service LookupProvider) LookupCodes(path string) []form.LookupCode {

	switch path {

	case "sharing":
		return []form.LookupCode{
			{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			{Value: "authenticated", Label: "Authenticated People Only"},
			{Value: "private", Label: "Only Selected Groups"},
		}

	case "groups":
		return service.Group.ListAsOptions()
	}

	return []form.LookupCode{}
}
