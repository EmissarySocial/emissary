// old imports
import m from "mithril"
import stream from "mithril/stream"
import {type ClientConfig, type KeyPackage, type Welcome} from "ts-mls"
import {MLS} from "./service/mls"
import type {Config} from "./model/config"
import {MLSFactory} from "./service/mls-factory"
import {type APActor} from "./model/ap-actor"
import {NewConfig} from "./model/config"
import type {Delivery} from "./service/delivery"
import type {Directory} from "./service/directory"
import type {Database} from "./service/database"
import type {Receiver} from "./service/receiver"
import type {Message} from "./model/message"
import {type Group} from "./model/group"

export class Controller {
	#actor: APActor
	#database: Database
	#delivery: Delivery
	#directory: Directory
	#receiver: Receiver
	#mls?: MLS
	config: Config
	clientConfig: ClientConfig
	selectedGroupId: string
	groups: stream<Group[]>
	messages: stream<Message[]>

	// constructor initializes the Controller with its dependencies
	constructor(
		actor: APActor,
		database: Database,
		delivery: Delivery,
		directory: Directory,
		receiver: Receiver,
		clientConfig: ClientConfig,
	) {
		this.#actor = actor
		this.#database = database
		this.#delivery = delivery
		this.#directory = directory
		this.#receiver = receiver
		this.clientConfig = clientConfig
		this.selectedGroupId = ""
		this.groups = stream([] as Group[])
		this.messages = stream([] as Message[])

		// Application Configuration
		this.config = NewConfig() // Empty placeholder until loaded
		this.loadConfig()
		this.loadGroups()
	}

	//////////////////////////////////////////
	// Startup
	//////////////////////////////////////////

	// loadConfig retrieves the configuration from the
	// database and starts the MLS service (if encryption keys are present)
	async loadConfig() {
		this.config = await this.#database.loadConfig()

		if (this.config.hasEncryptionKeys) {
			this.startMLS()
		}
		m.redraw()
	}

	// startMLS initializes the MLS service IF the configuration includes encryption keys
	private async startMLS() {
		//
		// Guarantee dependency
		if (this.config.hasEncryptionKeys == false) {
			throw new Error("Cannot start MLS without encryption keys")
		}

		// Create the MLS instance
		this.#mls = await MLSFactory(
			this.#database,
			this.#delivery,
			this.#directory,
			this.#receiver,
			this.#actor,
			this.clientConfig,
			this.config.clientName,
		)

		// Wire UX redraws into database updates
		this.#database.onchange(() => {
			console.log("got onchange callback")
			this.loadGroups()
			this.loadMessages()
		})
	}

	// createEncryptionKeys creates a new set of encryption keys
	// for this user on this device
	async createEncryptionKeys(clientName: string, password: string, passwordHint: string) {
		//
		// Initialize the config
		this.config.ready = true
		this.config.welcome = true
		this.config.hasEncryptionKeys = true
		this.config.clientName = clientName
		this.config.password = password
		this.config.passwordHint = passwordHint

		// Save the config to IndexedDB
		await this.#database.saveConfig(this.config)

		// Start the MLS service
		this.startMLS()

		// Redraw the UX
		m.redraw()
	}

	// skipEncryptionKeys is called when the user just wants to
	// use "direct messages" and does not want to create encryption keys (yet)
	async skipEncryptionKeys() {
		//
		// Initialize the config
		this.config.welcome = true
		await this.#database.saveConfig(this.config)

		// Redraw the UX
		m.redraw()
	}

	//////////////////////////////////////////
	// Conversations (Plaintext)
	//////////////////////////////////////////

	// newConversation creates a new plaintext ActivityPub conversation
	// with the specified recipients
	async newConversation(to: string[], message: string) {
		//
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

	//////////////////////////////////////////
	// Groups (Encrypted)
	//////////////////////////////////////////

	// createGroup creates a new MLS-encrypted
	// group message with the specified recipients
	async createGroup(recipients: string[]): Promise<Group> {
		//
		// Guarantee dependency
		if (this.#mls == undefined) {
			throw new Error("MLS service is not initialized")
		}

		// Create a new group
		const group = await this.#mls.createGroup()

		// Add initial members to the group
		await this.#mls.addGroupMembers(group.id, recipients)

		// Update the selected group
		this.selectedGroupId = group.id

		// Reload groups and messages to refresh the UX
		await this.loadGroups()

		return group
	}

	// loadGroups retrieves all groups from the database and
	// updates the "groups" and "messages" streams.
	async loadGroups() {
		//
		// load groups from the database
		const groups = await this.#database.allGroups()

		// If there are no groups, then set all values to "empty" state
		if (groups.length == 0) {
			this.groups([])
			this.messages([])
			this.selectedGroupId = ""
			return
		}

		// Fall through means we have 1+ groups
		// If we don't have a group selected already, then "select" the first group
		if (this.selectedGroupId == "") {
			this.selectedGroupId = groups[0]!.id
		}

		// Set the groups and messages streams accordingly
		this.groups(groups)
		this.loadMessages()

		console.log(groups)
	}

	// selectGroup updates the "selectedGroupId" and reloads messages for that group
	selectGroup(groupId: string) {
		//
		// If this group is already selected, then do nothing
		if (groupId == this.selectedGroupId) {
			return
		}

		// Update the selected group and reload messages
		this.selectedGroupId = groupId
		this.loadMessages()
	}

	// saveGroup saves the specified group to the database and reloads groups
	async saveGroup(group: Group) {
		await this.#database.saveGroup(group)
		await this.loadGroups()
	}

	// deleteGroup deletes the specified group from the database
	async deleteGroup(group: string) {
		//
		// Guarantee dependency
		if (this.#database == undefined) {
			throw new Error("Database service is not initialized")
		}

		// Delete the group
		await this.#database.deleteGroup(group)
		await this.loadGroups()
	}

	//////////////////////////////////////////
	// Messages
	//////////////////////////////////////////

	// loadMessages retrieves all messages for the currently selected group and updates the "messages" stream
	async loadMessages() {
		const messages = await this.#database.allMessages(this.selectedGroupId)
		this.messages(messages)
		m.redraw()
	}

	// sendMessage sends a message to the specified group
	async sendMessage(message: string) {
		//
		// Guarantee dependencies
		if (this.#mls == undefined) {
			throw new Error("MLS service is not initialized")
		}

		if (this.selectedGroupId == "") {
			throw new Error("No group selected")
		}

		// Send the message to the group
		await this.#mls.sendGroupMessage(this.selectedGroupId, message)

		// Reload messages to refresh the UX
		this.loadMessages()
	}
}
