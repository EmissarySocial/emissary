import type {DBSchema, IDBPDatabase} from "idb/build/entry.js"
import {type DBGroup} from "../model/db-group"
import {openDB, deleteDB, wrap, unwrap} from "idb"

// Schema defines the layout of records stored in IndexedDB
interface Schema extends DBSchema {
	group: {
		key: string
		value: DBGroup
	}
}

export async function NewDatabase(): Promise<IDBPDatabase<Schema>> {
	return await openDB<Schema>("mls-database", undefined, {
		upgrade(db) {
			db.createObjectStore("group", {keyPath: "groupID"})
		},
	})
}

export class Database {
	#db: IDBPDatabase<Schema>

	constructor(db: IDBPDatabase<Schema>) {
		this.#db = db
	}

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
}
