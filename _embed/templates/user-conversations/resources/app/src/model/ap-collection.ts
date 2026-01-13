// ActivityPub Collection or OrderedCollection structure
export interface APCollection<T> {
	totalItems?: number
	items?: (string | T)[]
	orderedItems?: (string | T)[]
	first?: string
	next?: string
}

// APCCollectionHeader represents the header information for any AcivityPub collection.
// This is the expected output for the `keyPackages` property returned in Actors from the
// `/.api/actors` endpont
export interface APCollectionHeader {
	totalItems: number
	first: string
}

export function NewCollectionHeader() {
	return {
		totalItems: 0,
		first: "",
	}
}
