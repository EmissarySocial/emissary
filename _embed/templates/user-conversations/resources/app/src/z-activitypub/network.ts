import {myActorID, myOutboxID} from "./utils"

/**
 * load GETs an ActivityPub resource with proper Accept headers.
 * If a URL is provided, then it fetches the resource from the network.
 * If an object is provided, it simply returns it.
 *
 * @param url - The URL to fetch
 * @returns The parsed JSON response
 * @throws Error if the fetch fails
 */
export async function load<T>(value: string): Promise<T> {
	//

	// If the URL is already an object, return it directly
	if (typeof value != "string") {
		return value
	}

	// Otherwise, the value is a URL, so load it from the network
	const response = await fetch(value, {
		headers: {
			Accept: 'application/activity+json, application/ld+json; profile="https://www.w3.org/ns/activitystreams"',
		},
	})

	if (!response.ok) {
		throw new Error(`Unable to fetch ${value}: ${response.status} ${response.statusText}`)
	}

	return response.json() as Promise<T>
}

// send POSTs an ActivityPub activity to the specified outbox
// and returns the Location header from the response
export async function send<T>(outbox: string, activity: T): Promise<string> {
	// Send the Activity to the server
	const response = await fetch(outbox, {
		method: "POST",
		body: JSON.stringify(activity),
		credentials: "include",
	})

	if (!response.ok) {
		throw new Error(`Failed to fetch ${outbox}: ${response.status} ${response.statusText}`)
	}

	return response.headers.get("Location") || ""
}

export async function createObject<T>(object: T): Promise<string> {
	return send(myOutboxID(), {
		"@context": "https://www.w3.org/ns/activitystreams",
		type: "Create",
		actor: myActorID(),
		object: object,
	})
}

export async function deleteObject(objectId: string): Promise<string> {
	return send(myOutboxID(), {
		"@context": "https://www.w3.org/ns/activitystreams",
		type: "Delete",
		actor: myActorID(),
		object: objectId,
	})
}
