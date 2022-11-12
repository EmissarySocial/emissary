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

	case "purgeDurations":
		return []form.LookupCode{
			{Label: "1 Day", Value: "1"},
			{Label: "1 Week", Value: "7"},
			{Label: "1 Month", Value: "31"},
			{Label: "1 Year", Value: "365"},
			{Label: "Forever", Value: "0"},
		}
	case "groups":
		return service.Group.ListAsOptions()
	}

	return []form.LookupCode{}
}
