import {createApplicationMessage} from "ts-mls"
import {createCommit} from "ts-mls"
import {createGroup} from "ts-mls"
import {joinGroup} from "ts-mls"
import {processPrivateMessage} from "ts-mls"
import {processPublicMessage} from "ts-mls"
import {getCiphersuiteFromName} from "ts-mls"
import {generateKeyPackage} from "ts-mls"
import {encodeMlsMessage} from "ts-mls"
import {decodeMlsMessage} from "ts-mls"
import {defaultCapabilities} from "ts-mls"
import {defaultLifetime} from "ts-mls"
import {emptyPskIndex} from "ts-mls"
import {nobleCryptoProvider} from "ts-mls"
import {type ClientState} from "ts-mls"
import {type Credential} from "ts-mls"
import {type Proposal} from "ts-mls"
import {type PrivateKeyPackage} from "ts-mls"
import {type KeyPackage} from "ts-mls"
import {type Welcome} from "ts-mls"
import {type PrivateMessage} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"

import type {APActor} from "../model/ap-actor"
import type {Database} from "./database"
import type {Delivery} from "./delivery"
import type {Directory} from "./directory"
import {MLS} from "./mls"

// makeMLS initializes the MLS service and returns it once all dependencies have been loaded
export async function NewMLS(
	database: Database,
	delivery: Delivery,
	directory: Directory,
	actor: APActor
): Promise<MLS> {
	//

	// Create a credential for this User
	const credential: Credential = {
		credentialType: "basic",
		identity: new TextEncoder().encode(actor.id),
	}

	// Use MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519 (ID: 1)
	// Using nobleCryptoProvider for compatibility (pure JS implementation)
	const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
	const cs = getCiphersuiteFromName(cipherSuiteName)
	const cipherSuite = await nobleCryptoProvider.getCiphersuiteImpl(cs)

	// Generate initial key package for this user
	const keyPackageResult = await generateKeyPackage(
		credential,
		defaultCapabilities(),
		defaultLifetime,
		[],
		cipherSuite
	)

	const publicKeyPackage = keyPackageResult.publicPackage
	const privateKeyPackage = keyPackageResult.privatePackage

	return new MLS(database, delivery, directory, actor, credential, cipherSuite, publicKeyPackage, privateKeyPackage)
}
