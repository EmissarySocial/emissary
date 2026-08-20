package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
)

// GroupLookupProvider lists this Domain's Groups as form lookup codes
type GroupLookupProvider struct {
	groupService *Group
	session      data.Session
}

// NewGroupLookupProvider returns a fully initialized GroupLookupProvider
func NewGroupLookupProvider(session data.Session, groupService *Group) GroupLookupProvider {
	return GroupLookupProvider{
		groupService: groupService,
		session:      session,
	}
}

// Get returns every Group on this Domain. Implements the form.LookupGroup interface.
func (service GroupLookupProvider) Get() []form.LookupCode {
	groups, _ := service.groupService.Query(service.session, exp.All())
	result := make([]form.LookupCode, 0, len(groups))

	for _, group := range groups {
		result = append(result, group.LookupCode())
	}

	return result

}

// Add creates a new Group with the provided name, and returns its ID. Implements the form.LookupGroup interface.
func (service GroupLookupProvider) Add(name string) (string, error) {

	group := model.NewGroup()
	group.Label = name

	if err := service.groupService.Save(service.session, &group, "created"); err != nil {
		return "", derp.Wrap(err, "service.GroupLookupProvider.Add", "Saving group", name)
	}

	return group.ID(), nil
}
