import {ActivityPubService} from "../z-activitypub/network"
import {KeyPackageService} from "./keyPackage"
import {type APActor} from "../z-activitypub/actor"

// Plaintext service manages unencrypted conversations
export class Plaintext {
	// All class #properties are PRIVATE
	#actor: APActor = {
		id: "",
		name: "",
		icon: "",
		username: "",
		inbox: "",
		mlsInbox: "",
		outbox: "",
		keyPackages: "",
	}

	#activityPub: ActivityPubService
	#keyPackage: KeyPackageService

	constructor() {
		this.#activityPub = new ActivityPubService()
		this.#keyPackage = new KeyPackageService(this.#activityPub)
	}

	async start() {
		const actor = await this.loadMyself()
		this.#actor = actor

		console.log(this.#actor)

		await this.#activityPub.start(actor.id)
		await this.#keyPackage.start(actor.id)
	}

	async loadMyself(): Promise<APActor> {
		// Retrieve my actor info from the server
		const response = await fetch("http://localhost/@me", {
			headers: [["Accept", "application/json"]],
		})

		// parse JSON
		const result = await response.json()

		// this MUST be an Actor value
		return result as APActor
	}

	async create(to: string[], message: string) {
		// Create an ActivityPub activity
		const activity = {
			"@context": "https://www.w3.org/ns/activitystreams",
			type: "Create",
			actor: this.#actor.id,
			to: to,
			object: {
				type: "Note",
				content: message,
			},
		}

		// POST to the actor's outbox
		const response = await fetch(this.#actor.outbox, {
			method: "POST",
			headers: {"Content-Type": "application/activity+json"},
			body: JSON.stringify(activity),
		})
	}
}
