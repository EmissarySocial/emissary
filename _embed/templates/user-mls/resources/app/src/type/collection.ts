
// APCollection is the ActivityPub representation of a Collection
// https://www.w3.org/TR/activitystreams-core/#collections
export type APCollection = {
	type: "Collection"
	id: string
	totalItems?: number
	items: string[]
}

// APOrderedCollection is the ActivityPub representation of an OrderedCollection
// https://www.w3.org/TR/activitystreams-core/#collections
export type APOrderedCollection = {
	type: "OrderedCollection"
	id: string
	totalItems?: number
	items: string[]
}
