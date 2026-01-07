import m from "mithril"

import {Main} from "./view/main"
import {MLSManager} from "./MLS/MLSManager"
import {type APActor} from "./activitypub/actor"
import {loadActor} from "./activitypub/actor"
import {getKeyPackages} from "./activitypub/keyPackage"

export class Controller {
	#actorId: string
	#actor: APActor | undefined

	constructor() {
		const root = document.getElementById("mls")

		if (root == undefined) {
			throw new Error("Can't mount Mithril app. Please verify that <div id=mls> exists.")
		}

		this.#actorId = root.dataset["actor-id"] || ""

		// Preload the actor
		this.getActor()
	}

	view() {
		return <Main controller={this} />
	}

	async getActor(): Promise<APActor> {
		// If we already have the actor cached, then just return it
		if (this.#actor != undefined) {
			return this.#actor
		}

		// Otherwise, load the actor from the server, cache and continue
		this.#actor = await loadActor(this.#actorId)
		return this.#actor
	}

	// newGroupAndMessage creates a new MLS-encrypted
	// group message with the specified recipients
	async newGroupAndMessage(recipients: string[], message: string) {
		const actor = new MLSManager(this.#actorId)
		const groupId = "1234567890"
		await actor.initialize()
		await actor.createGroup(groupId)

		// Get the KeyPackages for each recipient
		const keyPackages = await getKeyPackages(recipients)

		// Create the welcome message, ratchet tree, and commit
		const {welcome, ratchetTree, commit} = await actor.addMembers(groupId, keyPackages)

		// Update the rachet tree with the new members
		// Send the Welcome message to the group
		console.log("newGroupAndMessage", welcome, ratchetTree, commit)
	}

	// newConversation creates a new plaintext ActivityPub conversation
	// with the specified recipients
	async newConversation(to: string[], message: string) {
		// Create an ActivityPub activity
		const activity = {
			"@context": "https://www.w3.org/ns/activitystreams",
			type: "Create",
			actor: this.#actorId,
			to: to,
			object: {
				type: "Note",
				content: message,
			},
		}

		const actor = await this.getActor()

		// POST to the actor's outbox
		const response = await fetch(actor.outbox, {
			method: "POST",
			headers: {"Content-Type": "application/activity+json"},
			body: JSON.stringify(activity),
		})
	}
}

// Create and mount the main application
m.mount(document.getElementById("mls")!, Controller)
