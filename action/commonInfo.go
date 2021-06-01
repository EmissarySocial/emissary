package action

import "github.com/benpate/ghost/model"

type CommonInfo struct {
	ActionID string   `json:"actionId"`
	Method   string   `json:"method"`
	States   []string `json:"states"`
	Roles    []string `json:"roles"`
}

func NewCommonInfo(config *model.ActionConfig) CommonInfo {
	return CommonInfo{
		ActionID: config.ActionID,
		Method:   config.Method,
		States:   config.States,
		Roles:    config.Roles,
	}
}

// UserCan returns TRUE if this action is permitted on this stream (using the provided authorization)
func (info CommonInfo) UserCan(stream *model.Stream, authorization *model.Authorization) bool {

	if len(info.States) > 0 {
		if !matchOne(info.States, stream.StateID) {
			return false
		}
	}

	if len(info.Roles) > 0 {
		roles := stream.Roles(authorization)

		if !matchAny(roles, info.Roles) {
			return false
		}
	}

	return true
}

// matchOne returns TRUE if the value matches one (or more) of the values in the slice
func matchOne(slice []string, value string) bool {
	for index := range slice {
		if slice[index] == value {
			return true
		}
	}

	return false
}

// matchAny returns TRUE if any of the values in slice1 are equal to any of the values in slice2
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
