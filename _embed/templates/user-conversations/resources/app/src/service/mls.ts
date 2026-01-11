import {type APActor} from "../model/ap-actor"
import {type Group} from "../model/group"
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
import {stripTrailingNulls} from "./utils"
import {type ClientState} from "ts-mls"
import {type Credential} from "ts-mls"
import {type Proposal} from "ts-mls"
import {type PrivateKeyPackage} from "ts-mls"
import {type KeyPackage} from "ts-mls"
import type {MLSMessage} from "ts-mls/message.js"
import {type Welcome} from "ts-mls"
import {type PrivateMessage} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"
import {type ClientConfig} from "ts-mls"
import type {DBMessage} from "../model/db-message"

// IDatabase wraps all of the methods that the MLS service
// uses to store group state.
interface IDatabase {
	// load methods
	loadGroup(groupID: string): Promise<Group>
	loadMessage(messageID: string): Promise<DBMessage>

	// save methods
	saveGroup(group: Group): Promise<void>
	saveMessage(message: DBMessage): Promise<void>
}

// IDelivery wraps all of the methods that the MLS service
// uses to send messages.
interface IDelivery {
	sendWelcome(recipients: string[], welcome: Welcome): Promise<void>
	sendCommit(recipients: string[], commit: MLSMessage): Promise<void>
	sendPrivateMessage(recipients: string[], privateMessage: PrivateMessage): Promise<void>
}

// IDirectory wraps all of the methods that the MLS service
// uses to look up users' KeyPackages.
interface IDirectory {
	getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]>
}

// MLS service encrypts/decrypts messages using the MLS protocol
export class MLS {
	// Dependencies
	#database: IDatabase
	#delivery: IDelivery
	#directory: IDirectory
	#clientConfig: ClientConfig

	// Internal State
	#cipherSuite: CiphersuiteImpl
	#publicKeyPackage: KeyPackage
	#privateKeyPackage: PrivateKeyPackage
	#actor: APActor

	constructor(
		database: IDatabase,
		delivery: IDelivery,
		directory: IDirectory,
		actor: APActor,
		clientConfig: ClientConfig,
		cipherSuite: CiphersuiteImpl,
		publicKeyPackage: KeyPackage,
		privateKeyPackage: PrivateKeyPackage
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
	public async createGroup(): Promise<Group> {
		const groupID = crypto.randomUUID()
		const groupIDBytes = new TextEncoder().encode(groupID)

		// Create group using ts-mls
		const clientState = await createGroup(
			groupIDBytes,
			this.#publicKeyPackage!,
			this.#privateKeyPackage!,
			[],
			this.#cipherSuite,
			this.#clientConfig
		)

		// Populate a Group record
		const result: Group = {
			groupID: groupID,
			members: [this.#actor.id],
			name: "New Group",
			clientState: clientState,
			createDate: Date.now(),
			updateDate: Date.now(),
			readDate: Date.now(),
		}

		// Save the Group
		await this.#database.saveGroup(result)

		// Success
		return result
	}

	// addGroupMembers updates the group state.  It sends a Commit
	// message to existing members, and a Welcome message to new members,
	public async addGroupMembers(groupID: string, newMembers: string[]) {
		//

		console.log("mls.addGroupMembers: Adding members", newMembers, "to group", groupID)

		// load the group from the database
		const group = await this.#database.loadGroup(groupID)
		const currentMembers = group.members

		// Look up all KeyPackages for the new Members
		const keyPackages = await this.#directory.getKeyPackages(newMembers)

		console.log("mls.addGroupMembers: KeyPackages", keyPackages)

		// Create add proposals for each key package
		const addProposals: Proposal[] = keyPackages.map((keyPackage) => ({
			proposalType: "add",
			add: {
				keyPackage: keyPackage,
			},
		}))

		console.log("mls.addGroupMembers: Add Proposals", addProposals)

		// Create commit with add proposals
		const commitResult = await createCommit(
			{state: group.clientState, cipherSuite: this.#cipherSuite},
			{extraProposals: addProposals}
		)

		console.log("mls.addGroupMembers: Commit Result", commitResult)

		// (async) Send commit to existing members
		this.#delivery.sendCommit(currentMembers, commitResult.commit)

		// Send welcome to new members
		this.#delivery.sendWelcome(newMembers, commitResult.welcome!)

		// Update the group with new state and new list of members
		group.clientState = commitResult.newState
		group.members = currentMembers.concat(newMembers)
		await this.#database.saveGroup(group)
		console.log(group)

		// KEEPING THIS (DEAD?) CODE FOR NOW....
		// How will we use this rachet tree info??
		// Convert ratchetTree to a real array (it's Uint8Array-like with numeric indices)
		// const ratchetTreeArray = Array.from(commitResult.newState.ratchetTree)
		// RFC 9420: Strip trailing null nodes before transmission
		// const strippedTree = stripTrailingNulls(ratchetTreeArray)
	}

	public async sendGroupMessage(groupID: string, plaintext: string): Promise<void> {
		const message: DBMessage = {
			messageID: crypto.randomUUID(),
			groupID: groupID,
			senderID: this.#actor.id,
			plaintext: plaintext,
			ciphertext: new Uint8Array(),
			createDate: Date.now(),
		}
		await this.#database.saveMessage(message)
	}

	public encryptMessage(): string {
		return ""
	}
}

// debugging
function findNonSerializable(obj: any, path: string = "root"): string[] {
	const issues: string[] = []

	if (obj === null || typeof obj !== "object") {
		return issues
	}

	for (const key in obj) {
		if (obj.hasOwnProperty(key)) {
			const currentPath = `${path}.${key}`
			const value = obj[key]

			if (typeof value === "function") {
				issues.push(`${currentPath} is a function`)
			} else if (value instanceof Node) {
				issues.push(`${currentPath} is a DOM node`)
			} else if (typeof value === "symbol") {
				issues.push(`${currentPath} is a Symbol`)
			} else if (value instanceof Promise) {
				issues.push(`${currentPath} is a Promise`)
			} else if (value instanceof RegExp) {
				issues.push(`${currentPath} is a RegExp`)
			} else if (value instanceof Blob) {
				issues.push(`${currentPath} is a Blob`)
			} else if (typeof value === "object") {
				// Check for circular references
				try {
					structuredClone(value) // Modern way to test
				} catch (e) {
					issues.push(`${currentPath} has circular reference or non-serializable content`)
				}

				// Recursively check nested objects
				issues.push(...findNonSerializable(value, currentPath))
			}
		}
	}

	return issues
}
