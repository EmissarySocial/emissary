import {load} from "./network"

// ActivityPub Collection or OrderedCollection structure
interface APCollection<T> {
	totalItems?: number
	items?: (string | T)[]
	orderedItems?: (string | T)[]
	first?: string
	next?: string
}

// Async generator that fetches an ActivityPub collection and yields each item one by one.
// Automatically handles pagination by following 'first' and 'next' links.
//
// @param collectionUrl - The URL of the ActivityPub collection to fetch
// @yields Each item from the collection
export async function* rangeCollection<T>(url: string): AsyncGenerator<T> {
	//

	// Exit if if the URL is empty
	if (url == "") {
		return
	}

	// Fetch the collection object
	const collection = (await load(url)) as APCollection<T>

	// Yield any items that appear directly on the collection
	rangeCollectionPage<T>(collection)

	// Iterate on CollectionPages, starting with the "first" page
	var pageUrl = collection.first || collection.next

	while (pageUrl) {
		const page = (await load(pageUrl)) as APCollection<T>
		rangeCollectionPage(page)

		pageUrl = page.next
	}
}

// rangeCollectionPage yields all items from a CollectionPage or OrderedCollectionPage
async function* rangeCollectionPage<T>(collection: APCollection<T>): AsyncGenerator<T> {
	//

	// Get items array (could be 'items' or 'orderedItems')
	const items = collection.orderedItems || collection.items || []

	// Yield each item in the collection
	for (var item of items) {
		if (typeof item === "string") {
			item = (await load(item)) as T
		}

		yield item
	}
}
