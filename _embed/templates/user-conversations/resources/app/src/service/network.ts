import {type APCollection} from "../model/ap-collection"

// This file provides tools for retrieving documents from the network

// loadActivityStream fetches a single ActivityStream/JSON-LD document from the network.
export async function loadActivityStream(url: string): Promise<any> {
	//

	// Standard request headers for ActivityPub
	const headers = {
		Accept: 'application/activity+json, application/ld+json; profile="https://www.w3.org/ns/activitystreams"',
	}

	// Load the document from the network
	const response = await fetch(url, {headers: headers})

	// Annotate errors
	if (!response.ok) {
		throw new Error(`Unable to fetch ${url}: ${response.status} ${response.statusText}`)
	}

	// Parse and return the JSON response
	return await response.json()
}

// Async generator that fetches an ActivityPub collection and yields each item one by one.
// Automatically handles pagination by following 'first' and 'next' links.
export async function* rangeCollection<T>(url: string): AsyncGenerator<T> {
	//
	console.log("rangeCollection: fetching collection from URL:", url)
	// Exit if if the URL is empty
	if (url == "") {
		return
	}

	// Fetch the collection object
	const collection = (await loadActivityStream(url)) as APCollection<T>

	// If items are embedded directly in the page, then just return those
	if (collection.items || collection.orderedItems) {
		for await (const item of rangeCollectionPage<T>(collection)) {
			yield item
		}
		return
	}

	// Iterate on CollectionPages, starting with the "first" page
	var pageUrl = collection.first || collection.next

	while (pageUrl) {
		const page = (await loadActivityStream(pageUrl)) as APCollection<T>
		for await (const item of rangeCollectionPage<T>(page)) {
			yield item
		}

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
			item = (await loadActivityStream(item)) as T
		}

		yield item
	}
}
