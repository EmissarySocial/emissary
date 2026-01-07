
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

export type Article = {
	type: "Article"
	id: string
	content: string
}

export type Audio = {
	type: "Audio"
	id: string
	content: string
}

export type Image = {
	type: "Image"
	id: string
	content: string
}

export type Note = {
	type: "Note"
	id: string
	content: string
}

export type Video = {
	type: "Video"
	id: string
	content: string
}

