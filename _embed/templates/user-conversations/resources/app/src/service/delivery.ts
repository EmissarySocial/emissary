// import {type MLSMessage} from "ts-mls/message.js"
import {bytesToBase64} from "ts-mls"
import {encode} from "ts-mls"
import {type Commit} from "ts-mls"
import {type MlsPrivateMessage} from "ts-mls"
import {type MlsWelcome} from "ts-mls"
import {mlsPrivateMessageEncoder, type MlsFramedMessage} from "ts-mls/message.js"
import {mlsWelcomeEncoder} from "ts-mls/message.js"
import {mlsMessageEncoder} from "ts-mls/message.js"

// Delivery service sends messages via ActivityPub
export class Delivery {
	// context is the default JSON-LD context for MLS messages
	#context = ["https://www.w3.org/ns/activitystreams", "https://purl.archive.org/socialweb/mls"]

	// actorId is the ID of the user sending messages
	#actorId: string

	// outboxUrl is the URL of the user's outbox
	#outboxUrl: string

	constructor(actorId: string, outboxUrl: string) {
		this.#actorId = actorId
		this.#outboxUrl = outboxUrl
	}

	/**
	 * load GETs an ActivityPub resource with proper Accept headers.
	 * If a URL is provided, then it fetches the resource from the network.
	 * If an object is provided, it simply returns it.
	 *
	 * @param url - The URL to fetch
	 * @returns The parsed JSON response
	 * @throws Error if the fetch fails
	 */
	async load<T>(url: string): Promise<T> {
		//

		// If the URL is already an object, return it directly
		if (typeof url != "string") {
			return url
		}

		// Otherwise, the url is a URL, so load it from the network
		const response = await fetch(url, {
			headers: {
				Accept: 'application/activity+json, application/ld+json; profile="https://www.w3.org/ns/activitystreams"',
			},
		})

		if (!response.ok) {
			throw new Error(`Unable to fetch ${url}: ${response.status} ${response.statusText}`)
		}

		return response.json() as Promise<T>
	}

	// sendCommit sends an MLS commit message to the specified recipients
	async sendCommit(recipients: string[], commit: MlsFramedMessage) {
		//
		// Encode the commit message as JSON, then to bytes
		const content = encode(mlsMessageEncoder, commit)

		// Create an ActivityPub activity for the commit message
		const activity = {
			"@context": this.#context,
			type: "Create",
			actor: this.#actorId,
			to: recipients,
			object: {
				type: "mls:PrivateMessage",
				to: recipients,
				mediaType: "message/mls",
				encoding: "base64",
				content: bytesToBase64(content),
			},
		}

		// Send the activity
		await this.send(this.#outboxUrl, activity)
	}

	// sendWelcome sends an MLS welcome message to the specified recipients
	async sendWelcome(recipients: string[], welcome: MlsWelcome) {
		const content = bytesToBase64(encode(mlsWelcomeEncoder, welcome))

		const activity = {
			"@context": this.#context,
			type: "Create",
			actor: this.#actorId,
			to: recipients,
			object: {
				type: "mls:Welcome",
				to: recipients,
				mediaType: "message/mls",
				encoding: "base64",
				content: content,
			},
		}

		await this.send(this.#outboxUrl, activity)
	}

	// sendMessage sends an MLS private message to the specified recipients
	async sendMessage(recipients: string[], message: MlsFramedMessage) {
		const content = bytesToBase64(encode(mlsMessageEncoder, message))

		const activity = {
			"@context": this.#context,
			type: "Create",
			actor: this.#actorId,
			to: recipients,
			object: {
				type: "mls:PrivateMessage",
				to: recipients,
				mediaType: "message/mls",
				encoding: "base64",
				content: content,
			},
		}

		await this.send(this.#outboxUrl, activity)
	}

	// send POSTs an ActivityPub activity to the specified outbox
	// and returns the Location header from the response
	async send<T>(outbox: string, activity: T): Promise<string> {
		//

		// Send the Activity to the server
		const response = await fetch(outbox, {
			method: "POST",
			body: JSON.stringify(activity),
			credentials: "include",
		})

		if (!response.ok) {
			throw new Error(`Failed to POST ${outbox}: ${response.status} ${response.statusText}`)
		}

		return response.headers.get("Location") || ""
	}
}
