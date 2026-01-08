// ActivityPub Collection or OrderedCollection structure
export interface APCollection<T> {
	totalItems?: number
	items?: (string | T)[]
	orderedItems?: (string | T)[]
	first?: string
	next?: string
}
