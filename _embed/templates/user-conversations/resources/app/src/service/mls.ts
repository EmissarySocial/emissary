import {type APActor} from "../model/ap-actor"
import {type DBGroup} from "../model/db-group"
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
import type {MLSMessage} from "ts-mls/message.js"
import {type Welcome} from "ts-mls"
import {type PrivateMessage} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"
import {stripTrailingNulls} from "./utils"
import type {DBMessage} from "../model/db-message"

// IDatabase wraps all of the methods that the MLS service
// uses to store group state.
interface IDatabase {
	// load methods
	loadGroup(groupID: string): Promise<DBGroup>
	loadMessage(messageID: string): Promise<DBMessage>

	// save methods
	saveGroup(group: DBGroup): Promise<void>
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

	// Internal State
	#cipherSuite: CiphersuiteImpl
	#credential: Credential
	#publicKeyPackage: KeyPackage
	#privateKeyPackage: PrivateKeyPackage
	#actor: APActor

	constructor(
		database: IDatabase,
		delivery: IDelivery,
		directory: IDirectory,
		actor: APActor,
		credential: Credential,
		cipherSuite: CiphersuiteImpl,
		publicKeyPackage: KeyPackage,
		privateKeyPackage: PrivateKeyPackage
	) {
		this.#database = database
		this.#delivery = delivery
		this.#directory = directory
		this.#actor = actor
		this.#credential = credential
		this.#cipherSuite = cipherSuite
		this.#publicKeyPackage = publicKeyPackage
		this.#privateKeyPackage = privateKeyPackage
	}

	// createGroup creates a new MLS group and saves it to the database
	public async createGroup(): Promise<DBGroup> {
		const groupID = crypto.randomUUID()
		const groupIDBytes = new TextEncoder().encode(groupID)

		// Create group using ts-mls
		const groupState = await createGroup(
			groupIDBytes,
			this.#publicKeyPackage!,
			this.#privateKeyPackage!,
			[],
			this.#cipherSuite!
		)

		// Populate a DBGroup record
		const result: DBGroup = {
			groupID: groupID,
			members: [this.#actor.id],
			name: "New Group",
			groupState: groupState,
			createDate: Date.now(),
			updateDate: Date.now(),
			readDate: Date.now(),
		}

		// Save the DBGroup
		await this.#database.saveGroup(result)

		// Success
		return result
	}

	public async addGroupMembers(groupID: string, newMembers: string[]) {
		//

		// load the group from the database
		const group = await this.#database.loadGroup(groupID)
		const currentMembers = group.members

		// Look up all KeyPackages for the new Members
		const keyPackages = await this.#directory.getKeyPackages(newMembers)

		// Create add proposals for each key package
		const addProposals: Proposal[] = keyPackages.map((keyPackage) => ({
			proposalType: "add",
			add: {
				keyPackage: keyPackage,
			},
		}))

		// Create commit with add proposals
		const commitResult = await createCommit(
			{state: group.groupState, cipherSuite: this.#cipherSuite},
			{extraProposals: addProposals}
		)

		// (async) Send commit to existing members
		this.#delivery.sendCommit(currentMembers, commitResult.commit)

		// Send welcome to new members
		this.#delivery.sendWelcome(newMembers, commitResult.welcome!)

		// Update the group with new state and new list of members
		group.groupState = commitResult.newState
		group.members = currentMembers.concat(newMembers)
		await this.#database.saveGroup(group)

		/*
		// Debug: Log the commit structure
		console.group("🔍 [MLS Debug] Commit Structure")

		// RFC 9420 Section 11.2: Commit Distribution
		// ⚠️ IMPORTANT: The returned commit MUST be sent to all existing group members
		// so they can process it with processCommit() to stay synchronized.
		//
		// Distribution flow:
		// 1. Alice adds Bob: addMembers() returns { welcome, commit }
		// 2. Alice sends welcome to Bob (new member)
		// 3. Alice sends commit to existing members (Charlie, David, etc.)
		// 4. All existing members call processCommit(commit) to update their state
		//
		// Without distributing the commit, existing members will remain at old epoch
		// and won't be able to decrypt messages from the updated group.

		// Convert ratchetTree to a real array (it's Uint8Array-like with numeric indices)
		const ratchetTreeArray = Array.from(commitResult.newState.ratchetTree)
		// RFC 9420: Strip trailing null nodes before transmission
		const strippedTree = stripTrailingNulls(ratchetTreeArray)

		return {
			welcome: commitResult.welcome,
			ratchetTree: strippedTree,
			commit: commitResult.commit,
		}
		*/
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
