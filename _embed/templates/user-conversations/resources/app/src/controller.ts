// old imports
import m from "mithril"
import {type KeyPackage, type Welcome} from "ts-mls"
import {type MLSMessage} from "ts-mls/message.js"
import {type APActor} from "./model/ap-actor"
import {type Group} from "./model/group"
import {MLS} from "./service/mls"
import {Main} from "./view/main"

// IDatabase wraps all of the methods that the MLS service
// uses to store group state.
interface IDatabase {
	saveGroup(group: Group): Promise<void>
	loadGroup(groupID: string): Promise<Group>
}

// IDelivery wraps all of the methods that the MLS service
// uses to send messages.
interface IDelivery {
	sendWelcome(recipients: string[], welcome: Welcome): Promise<void>
	sendCommit(recipients: string[], commit: MLSMessage): Promise<void>
}

// IDirectory wraps all of the methods that the MLS service
// uses to look up users' KeyPackages.
interface IDirectory {
	getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]>
}

export class Controller {
	#actor: APActor

	#database: IDatabase
	#delivery: IDelivery
	#directory: IDirectory
	#mls: MLS

	// constructor initializes the Controller with its dependencies
	constructor(actor: APActor, database: IDatabase, delivery: IDelivery, directory: IDirectory, mls: MLS) {
		this.#actor = actor
		this.#database = database
		this.#delivery = delivery
		this.#directory = directory
		this.#mls = mls
	}

	// newGroupAndMessage creates a new MLS-encrypted
	// group message with the specified recipients
	async newGroupAndMessage(recipients: string[], message: string) {
		const group = await this.#mls.createGroup()
		await this.#mls.addGroupMembers(group.groupID, recipients)
	}

	// newConversation creates a new plaintext ActivityPub conversation
	// with the specified recipients
	async newConversation(to: string[], message: string) {
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
