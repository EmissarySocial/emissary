package model

import (
	"encoding/json"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// ActionConfig stores the configuration information for each Action that can be taken on a Stream
type ActionConfig struct {
	ActionID string         `json:"actionId"`
	Method   string         `json:"method"`
	States   []string       `json:"states"`
	Roles    []string       `json:"roles"`
	Then     []ActionConfig `json:"then"`

	datatype.Map `json:"args"`
}

// NewActionConfig returns a fully initialized ActionConfig object
func NewActionConfig() ActionConfig {
	return ActionConfig{
		States: make([]string, 0),
		Roles:  make([]string, 0),
		Then:   make([]ActionConfig, 0),
		Map:    datatype.NewMap(),
	}
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (actionConfig *ActionConfig) UnmarshalJSON(data []byte) error {

	actionConfig.Map = make(datatype.Map)

	dataMap := make(map[string]interface{})

	if err := json.Unmarshal(data, &dataMap); err != nil {
		return err
	}

	return actionConfig.UnmarshalMap(dataMap)
}

// UnmarshalMap copies data from a generic map into this ActionConfig
func (actionConfig *ActionConfig) UnmarshalMap(dataMap map[string]interface{}) error {

	for key, value := range dataMap {
		switch key {
		case "actionId":
			actionConfig.ActionID = convert.String(value)
		case "method":
			actionConfig.Method = convert.String(value)
		case "states":
			actionConfig.States = convert.SliceOfString(value)
		case "roles":
			actionConfig.Roles = convert.SliceOfString(value)
		case "then":
			then, err := convertSliceOfActionConfig(value)
			if err != nil {
				return derp.Wrap(err, "ghost.model.ActionConfig.UnmarshalMap", "Error processing 'then' properties")
			}
			actionConfig.Then = then

		default:
			actionConfig.Map[key] = value
		}
	}

	return nil
}

// convertSliceOfActionConfig attempts to generate a slice of action
func convertSliceOfActionConfig(data interface{}) ([]ActionConfig, error) {

	// Empty data sets are okay.
	if data == nil {
		return make([]ActionConfig, 0), nil
	}

	// Try to convert the data into an array/slice
	if sliceOfInterface, ok := data.([]interface{}); ok {

		result := make([]ActionConfig, len(sliceOfInterface))

		for index, value := range sliceOfInterface {

			result[index] = NewActionConfig()

			if dataMap, ok := value.(map[string]interface{}); ok {

				if err := result[index].UnmarshalMap(dataMap); err != nil {
					return nil, derp.Wrap(err, "ghost.model.convertSliceOfActionConfig", "Error unmarshalling map")
				}
			}
		}

		return result, nil
	}

	// Fall through means this is not a slice
	return nil, derp.New(derp.CodeBadRequestError, "ghost.model.convertSliceOfActionConfig", "Invalid type", data)
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (actionConfig ActionConfig) UserCan(stream *Stream, authorization *Authorization) bool {

	// If present, "States" limits the states where this action can take place
	if len(actionConfig.States) > 0 {
		// If states are present, then the current state MUST be included in the list.
		// Otherwise, reject this action.
		if !matchOne(actionConfig.States, stream.StateID) {
			return false
		}
	}

	// If present, "Roles" limits the user roles that can take this action
	if len(actionConfig.Roles) > 0 {

		// The user must have AT LEAST ONE of the named roles to take this action.
		// If not, reject this action.
		roles := stream.Roles(authorization)

		if !matchAny(roles, actionConfig.Roles) {
			return false
		}
	}

	// All filters have passed.  Allow this action.
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
