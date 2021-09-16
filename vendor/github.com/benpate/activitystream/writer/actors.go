package writer

import "github.com/benpate/activitystream/vocabulary"

func Application(name string, language string) Object {
	return NewObject().
		Type(vocabulary.ActorTypeApplication).
		Name(name, language)
}

func Group(name string, language string) Object {
	return NewObject().
		Type(vocabulary.ActorTypeGroup).
		Name(name, language)
}

func Organization(name string, language string) Object {
	return NewObject().
		Type(vocabulary.ActorTypeOrganization).
		Name(name, language)
}

func Person(name string, language string) Object {
	return NewObject().
		Type(vocabulary.ActorTypePerson).
		Name(name, language)
}

func Service(name string, language string) Object {
	return NewObject().
		Type(vocabulary.ActorTypeService).
		Name(name, language)
}
