import {type KeyPackage} from "ts-mls"
import {decodeMlsMessage} from "ts-mls"
import {type APActor} from "../model/ap-actor"
import {type APKeyPackage} from "../model/ap-keypackage"
import {loadActivityStream} from "./network"
import {rangeCollection} from "./network"
import {base64ToUint8Array} from "./utils"

export class Directory {
	#actorID: string // ID of the local actor
	#outboxURL: string // Outbox URL of the local actor

	constructor(actorID: string, outboxURL: string) {
		this.#actorID = actorID
		this.#outboxURL = outboxURL
	}

	// getKeyPackage loads the KeyPackages published by a single actor
	async getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]> {
		var result: KeyPackage[] = []

		for (const actorID of actorIDs) {
			console.log("getKeyPackages: Loading KeyPackages for: " + actorID)
			const actor = (await loadActivityStream(actorID)) as APActor
			console.log(actor)
			const rangeKeyPackages = rangeCollection<APKeyPackage>(actor.keyPackages)

			for await (const item of rangeKeyPackages) {
				console.log("KeyPackage item", item)
				const contentBytes = base64ToUint8Array(item.content)
				const mlsMessage = decodeMlsMessage(contentBytes, 0)![0]

				if (mlsMessage.wireformat != "mls_key_package") {
					throw new Error("Invalid KeyPackage message")
				}

				result.push(mlsMessage.keyPackage)
			}
		}

		return result
	}

	// createKeyPackage publishes a new KeyPackage to the User's outbox.
	async createKeyPackage(keyPackage: APKeyPackage): Promise<string> {
		return await this.createObject<APKeyPackage>(keyPackage)
	}

	// createObject POSTs an ActivityPub object to the user's outbox
	// and returns the Location header from the response
	private async createObject<T>(object: T): Promise<string> {
		return await this.send(this.#outboxURL, {
			"@context": "https://www.w3.org/ns/activitystreams",
			type: "Create",
			actor: this.#actorID,
			object: object,
		})
	}

	// send POSTs an ActivityPub activity to the specified outbox
	// and returns the Location header from the response
	private async send<T>(outbox: string, activity: T): Promise<string> {
		// Send the Activity to the server
		const response = await fetch(outbox, {
			method: "POST",
			body: JSON.stringify(activity),
			credentials: "include",
		})

		console.log("directory.send", response)

		if (!response.ok) {
			throw new Error(`Failed to fetch ${outbox}: ${response.status} ${response.statusText}`)
		}

		return response.headers.get("Location") || ""
	}
}
