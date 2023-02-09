package activitystreams

type Object map[string]any

func NewObject() Object {
	return map[string]any{
		"@context": DefaultContext(),
	}
}
