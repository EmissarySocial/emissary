import type {DBSchema, IDBPDatabase} from "idb/build/entry.js"
import {openDB} from "idb"
import {type DBGroup} from "../model/db-group"
import {type DBMessage} from "../model/db-message"
import type {DBKeyPackage} from "../model/db-keypackage"

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

	constructor(db: IDBPDatabase<Schema>) {
		this.#db = db
	}

	/////////////////////////////////////////////
	// Groups
	/////////////////////////////////////////////

	// saveGroup saves a group to the database
	async saveGroup(group: DBGroup) {
		await this.#db.put("group", group)
	}

	// loadGroup retrieves a group from the database
	async loadGroup(groupID: string): Promise<DBGroup> {
		const group = await this.#db.get("group", groupID)
		if (group == undefined) {
			throw new Error("Group not found: " + groupID)
		}
		return group
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
}
