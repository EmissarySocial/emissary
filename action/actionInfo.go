package action

import (
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/model"
)

type Info struct {
	ActionID string
	States   []string
	Roles    []string
}

// Returns TRUE if the request is authorized to execute this action with the provided stream
func (info Info) UserCan(request *domain.HTTPRequest, stream *model.Stream) bool {

	if len(info.States) > 0 {
		if !matchOne(info.States, stream.StateID) {
			return false
		}
	}

	if len(info.Roles) > 0 {

		authorization := request.Authorization()
		roles := stream.Roles(authorization)

		if !matchAny(roles, info.Roles) {
			return false
		}
	}

	return true
}

func matchOne(slice []string, value string) bool {
	for index := range slice {
		if slice[index] == value {
			return true
		}
	}

	return false
}

func matchAny(slice1 []string, slice2 []string) bool {

	for index1 := range slice1 {
		for index2 := range slice2 {
			if slice1[index1] == slice2[index2] {
				return true
			}
		}
	}

	return false
}
