import {type MlsGroupInfo, type MlsMessageProtocol} from "ts-mls"
import {type MlsFramedMessage} from "ts-mls"
import {type MlsPrivateMessage} from "ts-mls"
import {type MlsWelcomeMessage} from "ts-mls"

import {bytesToBase64, type Encoder} from "ts-mls"
import {encode} from "ts-mls"
import {decode} from "ts-mls"
import {mlsMessageEncoder} from "ts-mls"
import {mlsMessageDecoder} from "ts-mls"

// Delivery service sends messages via ActivityPub
export class Delivery {
	//

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

	// sendFramedMessage sends an MLS FramedMessage to the specified recipients
	sendFramedMessage(recipients: string[], message: MlsFramedMessage) {
		this.#send("mls:PrivateMessage", recipients, message, mlsMessageEncoder)
	}

	// sendGroupInfo sends an MLS GroupInfo message to the specified recipients
	sendGroupInfo(recipients: string[], message: MlsGroupInfo & MlsMessageProtocol) {
		this.#send("mls:GroupInfo", recipients, message, mlsMessageEncoder)
	}

	// sendPrivateMessage sends an MLS PrivateMessage to the specified recipients
	sendPrivateMessage(recipients: string[], message: MlsFramedMessage) {
		this.#send("mls:PrivateMessage", recipients, message, mlsMessageEncoder)
	}

	// sendWelcome sends an MLS Welcome message to the specified recipients
	sendWelcome(recipients: string[], message: MlsWelcomeMessage) {
		this.#send("mls:Welcome", recipients, message, mlsMessageEncoder)
	}

	// #send is a private method that sends an MLS message via the user's ActivityPub outbox
	async #send<T>(type: string, recipients: string[], message: T, encoder: Encoder<T>) {
		//
		// Filter out "me" from the recipients list (we don't need to send the message to ourselves)
		const otherRecipients = recipients.filter((recipient) => recipient !== this.#actorId)

		// If there are no recipients to send to, just return early
		if (otherRecipients.length === 0) {
			return
		}

		// Encode the private message as bytes, then to base64
		const contentBytes = encode(encoder, message)
		const contentBase64 = bytesToBase64(contentBytes)

		const decodedMessage = decode(mlsMessageDecoder, contentBytes)
		console.log("Decoded message:", decodedMessage)

		// Create an ActivityPub activity for the private message
		const activity = {
			"@context": this.#context,
			type: "Create",
			actor: this.#actorId,
			to: otherRecipients,
			object: {
				type: type,
				to: otherRecipients,
				mediaType: "message/mls",
				encoding: "base64",
				content: contentBase64,
			},
		}

		// Send the Activity to the server
		const response = await fetch(this.#outboxUrl, {
			method: "POST",
			body: JSON.stringify(activity),
			credentials: "include",
		})

		if (!response.ok) {
			throw new Error(`Failed to POST ${this.#outboxUrl}: ${response.status} ${response.statusText}`)
		}
	}
}
