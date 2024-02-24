package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
)

type GroupLookupProvider struct {
	groupService *Group
}

func NewGroupLookupProvider(groupService *Group) GroupLookupProvider {
	return GroupLookupProvider{
		groupService: groupService,
	}
}

func (service GroupLookupProvider) Get() []form.LookupCode {
	groups, _ := service.groupService.Query(exp.All())
	result := make([]form.LookupCode, 0, len(groups))

	for _, group := range groups {
		result = append(result, group.LookupCode())
	}

	return result

}

func (service GroupLookupProvider) Add(name string) (string, error) {

	group := model.NewGroup()
	group.Label = name

	if err := service.groupService.Save(&group, "created"); err != nil {
		return "", derp.Wrap(err, "service.GroupLookupProvider.Add", "Error saving group", name)
	}

	return group.ID(), nil
}
