import m from "mithril"

import {defaultClientConfig} from "ts-mls/clientConfig.js"
import {type APActor} from "./model/ap-actor"
import {type Credential} from "ts-mls/credential.js"
import {Database, NewIndexedDB} from "./service/database"
import {Delivery} from "./service/delivery"
import {Directory} from "./service/directory"
import {loadActivityStream} from "./service/network"
import {MLSFactory} from "./service/mls-factory"
import {Controller} from "./controller"
import {Main} from "./view/main"
import {nobleCryptoProvider} from "ts-mls/crypto/implementation/noble/provider.js"
import {getCiphersuiteFromName} from "ts-mls/crypto/ciphersuite.js"
import {generateKeyPackage} from "ts-mls/keyPackage.js"
import {defaultLifetime} from "ts-mls/lifetime.js"
import {defaultCapabilities} from "ts-mls/defaultCapabilities.js"

// Global controller instance
var controller: Controller

async function startup() {
	// Collect arguments from the DOM
	const root = document.getElementById("mls")!
	const actorID = root.dataset["actor-id"] || ""

	// Verify that the root element exists
	if (root == undefined) {
		throw new Error(`Can't mount Mithril app. Please verify that <div id="mls"> exists.`)
	}

	// Load the actor object from the network
	const actor = (await loadActivityStream(actorID)) as APActor

	// Build dependencies
	const indexedDB = await NewIndexedDB()
	const database = new Database(indexedDB, defaultClientConfig)
	const delivery = new Delivery(actor.id, actor.outbox)
	const directory = new Directory(actor.id, actor.outbox)

	// Build the controller
	controller = new Controller(actor, database, delivery, directory, defaultClientConfig)

	// Pass the controller to the Main component and mount the main application
	m.mount(root, {view: () => <Main controller={controller} />})
}

// 3..2..1.. Go!
startup()
