import { openDB, deleteDB, wrap, unwrap, type IDBPDatabase, type IDBPObjectStore } from "idb";
import { type APKeyPackage, type IDBKeyPackage, NewAPKeyPackage } from "../type/keyPackage"

import { 
	type Credential,
	type KeyPackage,
	type PrivateKeyPackage,

	defaultCapabilities,
	defaultLifetime,
	generateKeyPackage, 
	getCiphersuiteImpl,
	getCiphersuiteFromName
} from "ts-mls"
import type { ActivityPubService } from "./activityPub";

// This is the KeyPackage service, that manages all interactions with KeyPackages in the
// local indexedDB database
export class KeyPackageService {

	// All class #properties are PRIVATE
	#activityPub: ActivityPubService
	#database: IDBPDatabase | undefined
	#keyPackages: IDBKeyPackage[] = []
	#actorID: string = ""

	constructor(activityPub: ActivityPubService) {
		this.#activityPub = activityPub
	}

	async start(actorID: string) {

		this.#actorID = actorID

		// Set up the database connection
		this.#database = await openDB("KeyPackage", 1, {
			upgrade: (db, oldVersion, _newVersion, transaction, event) => {

				if (oldVersion == 0) {
					var keyPackages = db.createObjectStore("KeyPackage")
					keyPackages.createIndex("KeyPackage_id", ["id"])
				}
			}
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
		const implementation = await getCiphersuiteImpl(getCiphersuiteFromName("MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"))

		const credential: Credential = {
			credentialType: "basic", 
			identity: new TextEncoder().encode("alice"),
		}

		var newPackage = await generateKeyPackage(
			credential, 
			defaultCapabilities(), 
			defaultLifetime, 
			[], 
			implementation,
		)

		// Create a new KeyPackage and send it to the Server
		var remotePackage = NewAPKeyPackage(this.#actorID, newPackage.publicPackage)
		var [remotePackage, err] = await this.#activityPub.createObject(remotePackage)

		if (err != "") {
			console.log(err)
			return 
		}

		console.log(remotePackage)

		// Create a new LOCAL record for this KeyPackage
		const localPackage: IDBKeyPackage = {
			id: remotePackage.id,
			privatePackage: newPackage.privatePackage,
			publicPackage: newPackage.publicPackage,
		}

		console.log(localPackage)

		// Add the KeyPackage to the IndexedDB
		this.#database.add("KeyPackage", localPackage, localPackage.id)
	}
}