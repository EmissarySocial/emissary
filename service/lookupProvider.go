package service

import (
	"github.com/benpate/form"
)

type LookupProvider struct {
	groupService *Group
}

func NewLookupProvider(groupService *Group) LookupProvider {
	return LookupProvider{
		groupService: groupService,
	}
}

func (service LookupProvider) Group(path string) form.LookupGroup {

	switch path {

	case "sharing":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Value: "anonymous", Label: "Everyone (including anonymous visitors)"},
			form.LookupCode{Value: "authenticated", Label: "Authenticated People Only"},
			form.LookupCode{Value: "private", Label: "Only Selected Groups"},
		)

	case "purgeDurations":
		return form.NewReadOnlyLookupGroup(
			form.LookupCode{Label: "1 Day", Value: "1"},
			form.LookupCode{Label: "1 Week", Value: "7"},
			form.LookupCode{Label: "1 Month", Value: "31"},
			form.LookupCode{Label: "1 Year", Value: "365"},
			form.LookupCode{Label: "Forever", Value: "0"},
		)

	case "groups":
		return NewGroupLookupProvider(service.groupService)

	default:
		return form.NewReadOnlyLookupGroup()
	}
}
