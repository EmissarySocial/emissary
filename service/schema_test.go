package service

import (
	"encoding/json"

	"github.com/benpate/derp"
	"github.com/qri-io/jsonschema"
)

func populateSchema(value string) (*jsonschema.Schema, *derp.Error) {

	result := jsonschema.Schema{}

	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, derp.New(500, "service.populateSchema", "Error reading Schema JSON", value, err)
	}

	return &result, nil
}
