import {openDB, deleteDB, wrap, unwrap, type IDBPDatabase, type IDBPObjectStore} from "idb"
import {type APKeyPackage, NewKeyPackage} from "../activitypub/keyPackage"
import {type IDBMLSKeyPackage} from "../model/mlsKeyPackage"
import {createObject} from "../activitypub/network"

import {
	type Credential,
	type KeyPackage,
	type PrivateKeyPackage,
	defaultCapabilities,
	defaultLifetime,
	generateKeyPackage,
	getCiphersuiteImpl,
	getCiphersuiteFromName,
} from "ts-mls"

// This is the KeyPackage service, that manages all interactions with KeyPackages in the
// local indexedDB database
export class KeyPackageService {
	// All class #properties are PRIVATE
	#database: IDBPDatabase | undefined
	#keyPackages: IDBMLSKeyPackage[] = []
	#actorID: string = ""

	constructor() {}

	async start(actorID: string) {
		this.#actorID = actorID

		// Set up the database connection
		this.#database = await openDB("KeyPackage", 1, {
			upgrade: (db, oldVersion, _newVersion, transaction, event) => {
				if (oldVersion == 0) {
					var keyPackages = db.createObjectStore("KeyPackage")
					keyPackages.createIndex("KeyPackage_id", ["id"])
				}
			},
		})

		// Load all KeyPackages from the IndexedDB
		const transaction = this.#database.transaction("KeyPackage", "readwrite")
		this.#keyPackages = await transaction.store.getAll()

		// IF empty, create/sync a new KeyPackage
		if (this.#keyPackages.length == 0) {
			this.createKeyPackage()
		}
	}

	// createKeyPackage creates a new KeyPackage and
	// synchronizes it with the server.
	private async createKeyPackage() {
		if (this.#database == undefined) {
			return
		}

		// Create a new KeyPackage
		const implementation = await getCiphersuiteImpl(
			getCiphersuiteFromName("MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519")
		)

		const credential: Credential = {
			credentialType: "basic",
			identity: new TextEncoder().encode("alice"),
		}

		var newPackage = await generateKeyPackage(
			credential,
			defaultCapabilities(),
			defaultLifetime,
			[],
			implementation
		)

		// Create a new KeyPackage and send it to the Server
		const remotePackage = NewKeyPackage(this.#actorID, newPackage.publicPackage)
		const remotePackageUrl = await createObject(remotePackage)

		if (remotePackageUrl == "") {
			throw new Error("Failed to create KeyPackage on server")
		}

		// Create a new LOCAL record for this KeyPackage
		const localPackage: IDBMLSKeyPackage = {
			id: remotePackageUrl,
			privatePackage: newPackage.privatePackage,
			publicPackage: newPackage.publicPackage,
		}

		// Add the KeyPackage to the IndexedDB
		this.#database.add("KeyPackage", localPackage, localPackage.id)
	}
}
