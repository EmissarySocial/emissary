import {defaultCredentialTypes} from "ts-mls"
import {getCiphersuiteFromName} from "ts-mls"
import {generateKeyPackage} from "ts-mls"
import {nobleCryptoProvider} from "ts-mls"
import {type Credential} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"
import {type ClientConfig} from "ts-mls"

import type {APActor} from "../model/ap-actor"
import type {Database} from "./database"
import type {Delivery} from "./delivery"
import type {Directory} from "./directory"
import type {Receiver} from "./receiver"
import {MLS} from "./mls"
import {NewAPKeyPackage} from "../model/ap-keypackage"

// makeMLS loads the required dependencies for the MLS service,
// and returns a fully populated MLS instance once everything is ready.
export async function MLSFactory(
	database: Database,
	delivery: Delivery,
	directory: Directory,
	receiver: Receiver,
	actor: APActor,
	clientConfig: ClientConfig,
	clientName: string,
): Promise<MLS> {
	//
	// Use MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519 (ID: 1)
	// Using nobleCryptoProvider for compatibility (pure JS implementation)
	const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
	const cipherSuite = await nobleCryptoProvider.getCiphersuiteImpl(getCiphersuiteFromName(cipherSuiteName))

	// Try to load the KeyPackage from the IndexedDB database
	var dbKeyPackage = await database.loadKeyPackage()

	// Create a new KeyPackage if none exists
	if (dbKeyPackage == undefined) {
		//
		// Create a credential for this User
		const credential: Credential = {
			credentialType: defaultCredentialTypes.basic,
			identity: new TextEncoder().encode(actor.id),
		}

		// Generate initial key package for this user
		var keyPackageResult = await generateKeyPackage({
			credential: credential,
			cipherSuite: cipherSuite,
		})

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
	var result = new MLS(
		database,
		delivery,
		directory,
		receiver,
		clientConfig,
		cipherSuite,
		dbKeyPackage.publicKeyPackage,
		dbKeyPackage.privateKeyPackage,
		actor,
	)

	// Wire the receiver into the MLS service so that incoming messages are processed
	receiver.registerHandler(result.onMessage)

	// Start the receiver
	receiver.start()

	return result
}
