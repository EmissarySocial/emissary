package model

import (
	"encoding/json"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
)

// ActionConfig stores the configuration information for each Action that can be taken on a Stream
type ActionConfig struct {
	ActionID string       `json:"actionId"`
	Method   string       `json:"method"`
	States   []string     `json:"states"`
	Roles    []string     `json:"roles"`
	Args     datatype.Map `json:"args"`
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (actionConfig ActionConfig) UnmarshalJSON(data []byte) error {

	actionConfig.Args = make(datatype.Map)

	dataMap := make(map[string]interface{})

	if err := json.Unmarshal(data, &dataMap); err != nil {
		return err
	}

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
		default:
			actionConfig.Args[key] = value
		}
	}

	return nil
}
