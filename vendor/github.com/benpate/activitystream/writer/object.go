package writer

import "github.com/benpate/activitystream/vocabulary"

func Article() Object {
	return NewObject().Type(vocabulary.ObjectTypeArticle)
}

func Audio() Object {
	return NewObject().Type(vocabulary.ObjectTypeAudio)
}

func Document() Object {
	return NewObject().Type(vocabulary.ObjectTypeDocument)
}

func Event() Object {
	return NewObject().Type(vocabulary.ObjectTypeEvent)
}

func Image() Object {
	return NewObject().Type(vocabulary.ObjectTypeImage)
}

func Note() Object {
	return NewObject().Type(vocabulary.ObjectTypeNote)
}

func Page() Object {
	return NewObject().Type(vocabulary.ObjectTypePage)
}

func Place() Object {
	return NewObject().Type(vocabulary.ObjectTypePlace)
}

func Profile() Object {
	return NewObject().Type(vocabulary.ObjectTypeProfile)
}

func Relationship() Object {
	return NewObject().Type(vocabulary.ObjectTypeRelationship)
}

func Tombstone() Object {
	return NewObject().Type(vocabulary.ObjectTypeTombstone)
}

func Video() Object {
	return NewObject().Type(vocabulary.ObjectTypeVideo)
}

func Mention() Object {
	return NewObject().Type(vocabulary.ObjectTypeMention)
}
