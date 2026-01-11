import type {DBSchema, IDBPDatabase} from "idb/build/entry.js"
import {openDB} from "idb"
import {type Group} from "../model/group"
import {type DBGroup} from "../model/db-group"
import {type DBMessage} from "../model/db-message"
import type {DBKeyPackage} from "../model/db-keypackage"
import {decodeGroupState, encodeGroupState} from "ts-mls"
import {type ClientConfig} from "ts-mls/clientConfig.js"
import {type ClientState} from "ts-mls"

// Schema defines the layout of records stored in IndexedDB
interface Schema extends DBSchema {
	group: {
		key: string
		value: DBGroup
	}

	keyPackage: {
		key: string
		value: DBKeyPackage
	}

	message: {
		key: string
		value: DBMessage
	}
}

export async function NewIndexedDB(): Promise<IDBPDatabase<Schema>> {
	return await openDB<Schema>("mls-database", undefined, {
		upgrade(db, oldVersion, newVersion) {
			console.log("Upgrading database from version", oldVersion, "to:", newVersion)
			db.createObjectStore("group", {keyPath: "groupID"})
			db.createObjectStore("keyPackage", {keyPath: "keyPackageID"})
			db.createObjectStore("message", {keyPath: "messageID"})
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
	// Groups
	/////////////////////////////////////////////

	// saveGroup saves a group to the database
	async saveGroup(group: Group) {
		// Encode the group (with serialized clientState)
		const dbGroup: DBGroup = {
			groupID: group.groupID,
			members: group.members,
			name: group.name,
			clientState: encodeGroupState(group.clientState),
			createDate: group.createDate,
			updateDate: group.updateDate,
			readDate: group.readDate,
		}

		const state = await this.#db.put("group", dbGroup)
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
			groupID: dbGroup.groupID,
			members: dbGroup.members,
			name: dbGroup.name,
			clientState: this.decodeClientState(dbGroup.clientState),
			createDate: dbGroup.createDate,
			updateDate: dbGroup.updateDate,
			readDate: dbGroup.readDate,
		}

		// Success?
		return result
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

	/////////////////////////////////////////////
	// Utilities
	/////////////////////////////////////////////

	decodeClientState(serialized: Uint8Array): ClientState {
		// Decode the group (with deserialized clientState)
		const decodedGroupState = decodeGroupState(serialized, 0)

		if (decodedGroupState == null) {
			throw new Error("Unable to decode group state")
		}

		var clientState: any = decodedGroupState[0]
		clientState.clientConfig = this.#clientConfig

		return clientState
	}
}
