package writer

import "github.com/benpate/activitystream/vocabulary"

type Property struct {
	name   string
	value  []string
	object *Object
}

func id(value string) Property {
	return Property{
		name:  vocabulary.PropertyID,
		value: []string{value},
	}
}
