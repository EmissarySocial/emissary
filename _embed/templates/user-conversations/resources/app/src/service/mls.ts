// MLS functions
import {createCommit, type MlsFramedMessage, type MlsPrivateMessage} from "ts-mls"
import {createGroup} from "ts-mls"
import {createApplicationMessage} from "ts-mls"
import {defaultProposalTypes} from "ts-mls"
import {unsafeTestingAuthenticationService} from "ts-mls"

// MLS Types
import {type Proposal} from "ts-mls"
import {type PrivateKeyPackage} from "ts-mls"
import {type KeyPackage} from "ts-mls"
import {type MlsContext} from "ts-mls"
import {type MlsWelcomeMessage} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"
import {type ClientConfig} from "ts-mls"

// Application Types
import {type APActor} from "../model/ap-actor"
import {type Group} from "../model/group"
import {type APKeyPackage} from "../model/ap-keypackage"
import {type DBMessage} from "../model/db-message"
import {type DBKeyPackage} from "../model/db-keypackage"

// IDatabase wraps all of the methods that the MLS service
// uses to store group state.
interface IDatabase {
	// load methods
	loadGroup(groupID: string): Promise<Group>
	loadMessage(messageID: string): Promise<DBMessage>

	// save methods
	saveGroup(group: Group): Promise<void>
	saveMessage(message: DBMessage): Promise<void>

	loadKeyPackage(): Promise<DBKeyPackage | undefined>
	saveKeyPackage(keyPackage: DBKeyPackage): Promise<void>
}

// IDelivery wraps all of the methods that the MLS service
// uses to send messages.
interface IDelivery {
	sendWelcome(recipients: string[], welcome: MlsWelcomeMessage): Promise<void>
	sendCommit(recipients: string[], commit: MlsFramedMessage): Promise<void>
	sendMessage(recipients: string[], message: MlsFramedMessage): Promise<void>
}

// IDirectory wraps all of the methods that the MLS service
// uses to look up users' KeyPackages.
interface IDirectory {
	getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]>
	createKeyPackage(keyPackage: APKeyPackage): Promise<string>
}

const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"

// MLS service encrypts/decrypts messages using the MLS protocol.
// This is intended to be a reusable service that could be called
// by any software component that needs to use MLS-encrypted messages.
export class MLS {
	#database: IDatabase
	#delivery: IDelivery
	#directory: IDirectory
	#clientConfig: ClientConfig
	#cipherSuite: CiphersuiteImpl
	#publicKeyPackage: KeyPackage
	#privateKeyPackage: PrivateKeyPackage
	#actor: APActor

	constructor(
		database: IDatabase,
		delivery: IDelivery,
		directory: IDirectory,
		clientConfig: ClientConfig,

		cipherSuite: CiphersuiteImpl,
		publicKeyPackage: KeyPackage,
		privateKeyPackage: PrivateKeyPackage,
		actor: APActor,
	) {
		this.#database = database
		this.#delivery = delivery
		this.#directory = directory
		this.#clientConfig = clientConfig

		this.#actor = actor
		this.#cipherSuite = cipherSuite
		this.#publicKeyPackage = publicKeyPackage
		this.#privateKeyPackage = privateKeyPackage
	}

	// createGroup creates a new MLS group and saves it to the database
	async createGroup(): Promise<Group> {
		const groupID = crypto.randomUUID()
		const groupIDBytes = new TextEncoder().encode(groupID)
		console.log("Creating group with ID:", groupID, groupIDBytes)
		console.log("context", this.#context())
		console.log("publicKeyPackage", this.#publicKeyPackage)
		console.log("privateKeyPackage", this.#privateKeyPackage)

		// Create group using ts-mls
		const clientState = await createGroup({
			context: this.#context(),
			groupId: groupIDBytes,
			keyPackage: this.#publicKeyPackage,
			privateKeyPackage: this.#privateKeyPackage,
		})

		// Populate a Group record
		const result: Group = {
			id: groupID,
			members: [this.#actor.id],
			name: "New Group",
			clientState: clientState,
			createDate: Date.now(),
			updateDate: Date.now(),
			readDate: Date.now(),
		}

		// Save the Group
		console.log("Saving group to database:", result)
		await this.#database.saveGroup(result)

		// Success
		return result
	}

	// addGroupMembers updates the group state.  It sends a Commit
	// message to existing members, and a Welcome message to new members,
	async addGroupMembers(groupID: string, newMembers: string[]) {
		//
		// load the group from the database
		const group = await this.#database.loadGroup(groupID)
		const currentMembers = group.members

		// Look up all KeyPackages for the new Members
		const keyPackages = await this.#directory.getKeyPackages(newMembers)

		// Create add proposals for each key package
		const addProposals: Proposal[] = keyPackages.map((keyPackage) => ({
			proposalType: defaultProposalTypes.add,
			add: {
				keyPackage: keyPackage,
			},
		}))

		// Create commit with add proposals
		const commitResult = await createCommit({
			context: this.#context(),
			state: group.clientState,
			extraProposals: addProposals,
		})

		// (async) Send commit to existing members
		this.#delivery.sendCommit(currentMembers, commitResult.commit)

		// Send welcome to new members
		this.#delivery.sendWelcome(newMembers, commitResult.welcome!)

		// Update the group with new state and new list of members
		group.clientState = commitResult.newState
		group.members = currentMembers.concat(newMembers)
		await this.#database.saveGroup(group)

		// KEEPING THIS (DEAD?) CODE FOR NOW....
		// How will we use this rachet tree info??
		// Convert ratchetTree to a real array (it's Uint8Array-like with numeric indices)
		// const ratchetTreeArray = Array.from(commitResult.newState.ratchetTree)
		// RFC 9420: Strip trailing null nodes before transmission
		// const strippedTree = stripTrailingNulls(ratchetTreeArray)
	}

	async sendGroupMessage(group: string, plaintext: string): Promise<void> {
		//
		// Encrypt the message using the current group state
		// (This is a placeholder - the actual encryption logic will depend on the ts-mls API)
		const mlsGroup = await this.#database.loadGroup(group)

		// Create the message object as JSON-LD
		const messageObject = {
			"@context": "https://www.w3.org/ns/activitystreams",
			type: "Note",
			content: plaintext,
		}

		// Encrypt the message using MLS
		const messageText = JSON.stringify(messageObject)
		const messageBytes = new TextEncoder().encode(messageText)
		const result = await createApplicationMessage({
			context: this.#context(),
			state: mlsGroup.clientState,
			message: messageBytes,
		})

		// Send the message via the Delivery service
		this.#delivery.sendMessage(mlsGroup.members, result.message)

		// update the Group with the new group state
		mlsGroup.clientState = result.newState
		mlsGroup.updateDate = Date.now()
		await this.#database.saveGroup(mlsGroup)

		// Create a new Message object
		const dbMessage: DBMessage = {
			id: crypto.randomUUID(),
			group: group,
			sender: this.#actor.id,
			plaintext: plaintext,
			createDate: Date.now(),
		}

		// Save the message to the database
		await this.#database.saveMessage(dbMessage)
	}

	encryptMessage(): string {
		return ""
	}

	#context(): MlsContext {
		return {
			cipherSuite: this.#cipherSuite,
			authService: unsafeTestingAuthenticationService,
		}
	}
}
