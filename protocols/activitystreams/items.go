package activitystreams

func UnmarshalItems(data any) ([]any, bool) {
	result, ok := data.([]any)
	return result, ok
}
