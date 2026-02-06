import type {DBSchema, IDBPDatabase} from "idb/build/entry.js"
import {openDB} from "idb"
import {ConfigID, NewConfig, type Config} from "../model/config"
import {type Group} from "../model/group"
import {type DBGroup} from "../model/db-group"
import {type DBMessage} from "../model/db-message"
import type {DBKeyPackage} from "../model/db-keypackage"
import {type ClientConfig} from "ts-mls/clientConfig.js"
import {type ClientState} from "ts-mls"

// Schema defines the layout of records stored in IndexedDB
interface Schema extends DBSchema {
	config: {
		key: string
		value: Config
		indexes: {
			id: string
		}
	}
	group: {
		key: string
		value: DBGroup
		indexes: {
			id: string
		}
	}

	keyPackage: {
		key: string
		value: DBKeyPackage
		indexes: {
			id: string
		}
	}

	message: {
		key: string
		value: DBMessage
		indexes: {
			id: string
			group: string
		}
	}
}

export async function NewIndexedDB(): Promise<IDBPDatabase<Schema>> {
	return await openDB<Schema>("mls-database", undefined, {
		upgrade(db, oldVersion, newVersion) {
			console.log("Upgrading database from version", oldVersion, "to:", newVersion)
			db.createObjectStore("config", {keyPath: "id"})
			db.createObjectStore("group", {keyPath: "id"})
			db.createObjectStore("keyPackage", {keyPath: "id"})

			const messages = db.createObjectStore("message", {keyPath: "id"})
			messages.createIndex("group", "group", {unique: false})
		},
	})
}

export class Database {
	#db: IDBPDatabase<Schema>
	#clientConfig: ClientConfig

	constructor(db: IDBPDatabase<Schema>, clientConfig: ClientConfig) {
		this.#db = db
		this.#clientConfig = clientConfig
	}

	/////////////////////////////////////////////
	// Config
	/////////////////////////////////////////////

	// loadConfig retrieves the config record from the database
	async loadConfig(): Promise<Config> {
		var result = await this.#db.get("config", ConfigID)
		if (result == undefined) {
			result = NewConfig()
		}

		// Mark this configuration as "loaded from the db"
		result.ready = true
		return result
	}

	// saveConfig saves the config record to the database
	async saveConfig(config: Config) {
		config.id = ConfigID
		config.ready = true
		await this.#db.put("config", config)
	}

	/////////////////////////////////////////////
	// Groups
	/////////////////////////////////////////////

	// allGroups returns all groups from the database, sorted by updateDate descending
	async allGroups(): Promise<DBGroup[]> {
		//
		// List all groups, sorted by updateDate descending
		var groups = await this.#db.getAll("group")
		groups.sort((a, b) => b.updateDate - a.updateDate)
		return groups
	}

	// saveGroup saves a group to the database
	async saveGroup(group: Group) {
		//
		// Encode the group (with serialized clientState)
		const dbGroup: DBGroup = {
			id: group.id,
			members: group.members,
			name: group.name,
			clientState: group.clientState,
			createDate: group.createDate,
			updateDate: group.updateDate,
			readDate: group.readDate,
		}

		await this.#db.put("group", dbGroup)
	}

	// loadGroup retrieves a group from the database
	async loadGroup(groupID: string): Promise<Group> {
		//

		// Load the group record
		const dbGroup = await this.#db.get("group", groupID)
		if (dbGroup == undefined) {
			throw new Error("Group not found: " + groupID)
		}

		// Create an in-memory group record
		const result: Group = {
			id: dbGroup.id,
			members: dbGroup.members,
			name: dbGroup.name,
			clientState: dbGroup.clientState,
			createDate: dbGroup.createDate,
			updateDate: dbGroup.updateDate,
			readDate: dbGroup.readDate,
		}

		// Success?
		return result
	}

	// deleteGroup removes a group from the database
	async deleteGroup(group: string) {
		//
		// List all messages in the group
		const messages = await this.#db.getAllKeysFromIndex("message", "group", group)

		// Delete messages in the group
		for (const message of messages) {
			await this.#db.delete("message", message)
		}

		// Delete the group itself
		await this.#db.delete("group", group)
	}

	/////////////////////////////////////////////
	// Private KeyPackage
	/////////////////////////////////////////////

	async loadKeyPackage() {
		const keyPackage = await this.#db.get("keyPackage", "self")
		return keyPackage
	}

	async saveKeyPackage(keyPackage: DBKeyPackage) {
		await this.#db.put("keyPackage", keyPackage)
	}

	/////////////////////////////////////////////
	// Messages
	/////////////////////////////////////////////

	// saveMessage saves a message to the database
	async saveMessage(message: DBMessage) {
		await this.#db.put("message", message)
	}

	// loadMessage retrieves a message from the database
	async loadMessage(messageID: string): Promise<DBMessage> {
		const message = await this.#db.get("message", messageID)
		if (message == undefined) {
			throw new Error("Message not found: " + messageID)
		}
		return message
	}

	async listMessages(group: string): Promise<DBMessage[]> {
		var messages = await this.#db.getAllFromIndex("message", "group", group)
		messages.sort((a, b) => a.createDate - b.createDate)
		return messages
	}
}
