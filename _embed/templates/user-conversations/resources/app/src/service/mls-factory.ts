import {getCiphersuiteFromName, getCiphersuiteImpl, type KeyPackage, type PrivateKeyPackage} from "ts-mls"
import {generateKeyPackage} from "ts-mls"
import {defaultCapabilities} from "ts-mls"
import {defaultLifetime} from "ts-mls"
import {nobleCryptoProvider} from "ts-mls"
import {type Credential} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"
import {type ClientConfig} from "ts-mls"

import type {APActor} from "../model/ap-actor"
import type {Database} from "./database"
import type {Delivery} from "./delivery"
import type {Directory} from "./directory"
import {MLS} from "./mls"
import {NewAPKeyPackage} from "../model/ap-keypackage"

// makeMLS loads the required dependencies for the MLS service,
// and returns a fully populated MLS instance once everything is ready.
export async function MLSFactory(
	database: Database,
	delivery: Delivery,
	directory: Directory,
	actor: APActor,
	clientConfig: ClientConfig,
	clientName: string
): Promise<MLS> {
	//

	console.log("MLSFactory: Starting MLS Factory")

	// Use MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519 (ID: 1)
	// Using nobleCryptoProvider for compatibility (pure JS implementation)
	const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
	const cipherSuite = await nobleCryptoProvider.getCiphersuiteImpl(getCiphersuiteFromName(cipherSuiteName))

	console.log("MLSFactory: loaded cipher suite", cipherSuiteName)

	// Try to load the KeyPackage from the IndexedDB database
	var dbKeyPackage = await database.loadKeyPackage()

	console.log("MLSFactory: loaded dbKeyPackage", dbKeyPackage)

	// Create a new KeyPackage if none exists
	if (dbKeyPackage == undefined) {
		//

		try {
			const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
			const cipherSuite = await nobleCryptoProvider.getCiphersuiteImpl(getCiphersuiteFromName(cipherSuiteName))

			// Create a credential for this User
			const credential: Credential = {
				credentialType: "basic",
				identity: new TextEncoder().encode(actor.id),
			}

			console.log("Generating Key package for actor:", actor)

			console.log(
				"Ima break??",
				generateKeyPackage,
				credential,
				defaultCapabilities,
				defaultLifetime,
				cipherSuite
			)
			// Generate initial key package for this user
			var keyPackageResult = await generateKeyPackage(
				credential,
				defaultCapabilities(),
				defaultLifetime,
				[],
				cipherSuite
			)
		} catch (error) {
			console.error("Error generating KeyPackage:", error)
			throw error
		}

		console.log("Generated Key package", keyPackageResult)

		// Publish the KeyPackage to the server
		const apKeyPackage = NewAPKeyPackage(clientName, actor.id, keyPackageResult.publicPackage)
		const apKeyPackageURL = await directory.createKeyPackage(apKeyPackage)

		if (apKeyPackageURL == "") {
			throw new Error("Failed to create KeyPackage on server")
		}

		// Save the KeyPackage to the local database
		dbKeyPackage = {
			id: "self",
			keyPackageURL: apKeyPackageURL,
			clientName: clientName,
			publicKeyPackage: keyPackageResult.publicPackage,
			privateKeyPackage: keyPackageResult.privatePackage,
			cipherSuiteName: cipherSuiteName,
		}

		await database.saveKeyPackage(dbKeyPackage)
	}

	// Create and return the MLS service
	return new MLS(
		database,
		delivery,
		directory,
		clientConfig,
		cipherSuite,
		dbKeyPackage.publicKeyPackage,
		dbKeyPackage.privateKeyPackage,
		actor
	)
}

// Use MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519 (ID: 1)
// Using nobleCryptoProvider for compatibility (pure JS implementation)
// Other implementations can be added in the future.
async function makeCipherSuite(
	cipherSuiteName: "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
): Promise<CiphersuiteImpl> {
	const cs = getCiphersuiteFromName(cipherSuiteName)
	return await nobleCryptoProvider.getCiphersuiteImpl(cs)
}
