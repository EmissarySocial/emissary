// old imports
import m from "mithril"
import {type ClientConfig, type KeyPackage, type Welcome} from "ts-mls"
import {type MLSMessage} from "ts-mls/message.js"
import {type APActor} from "./model/ap-actor"
import {type Group} from "./model/group"
import {MLS} from "./service/mls"
import type {Config} from "./model/config"
import {NewConfig} from "./model/config"
import {MLSFactory} from "./service/mls-factory"
import type {Delivery} from "./service/delivery"
import type {Directory} from "./service/directory"
import type {Database} from "./service/database"

export class Controller {
	#actor: APActor
	#database: Database
	#delivery: Delivery
	#directory: Directory
	#mls?: MLS
	config: Config
	clientConfig: ClientConfig

	// constructor initializes the Controller with its dependencies
	constructor(
		actor: APActor,
		database: Database,
		delivery: Delivery,
		directory: Directory,
		clientConfig: ClientConfig
	) {
		this.#actor = actor
		this.#database = database
		this.#delivery = delivery
		this.#directory = directory
		this.clientConfig = clientConfig

		// Application Configuration
		this.config = NewConfig() // Empty placeholder until loaded
		this.loadConfig()
	}

	async loadConfig() {
		this.config = await this.#database.loadConfig()

		if (this.config.hasEncryptionKeys) {
			this.startMLS()
		}
		m.redraw()
	}

	// skipEncryptionKeys is called when the user just wants to
	// use "direct messages" and does not want to create encryption keys (yet)
	async skipEncryptionKeys() {
		this.config.welcome = true
		await this.#database.saveConfig(this.config)
		m.redraw()
	}

	// createEncryptionKeys creates a new set of encryption keys
	// for this user on this device
	async createEncryptionKeys(clientName: string, password: string, passwordHint: string) {
		// TODO: Create encryption keys here...

		this.config.ready = true
		this.config.welcome = true
		this.config.hasEncryptionKeys = true
		this.config.clientName = clientName
		this.config.password = password
		this.config.passwordHint = passwordHint

		await this.#database.saveConfig(this.config)
		this.startMLS()

		m.redraw()
	}

	// newGroupAndMessage creates a new MLS-encrypted
	// group message with the specified recipients
	async newGroupAndMessage(recipients: string[], message: string) {
		if (this.#mls == undefined) {
			throw new Error("MLS service is not initialized")
		}

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

	// startMLS initializes the MLS service IF the configuration includes encryption keys
	private async startMLS() {
		if (this.config.hasEncryptionKeys == false) {
			throw new Error("Cannot start MLS without encryption keys")
		}

		this.#mls = await MLSFactory(
			this.#database,
			this.#delivery,
			this.#directory,
			this.#actor,
			this.clientConfig,
			this.config.clientName
		)
	}
}
