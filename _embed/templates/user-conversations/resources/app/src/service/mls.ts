// MLS functions
import {
	createCommit,
	defaultCredentialTypes,
	joinGroup,
	zeroOutUint8Array,
	type CredentialBasic,
	type MlsMessageProtocol,
} from "ts-mls"
import {wireformats} from "ts-mls"
import {createGroup} from "ts-mls"
import {createApplicationMessage} from "ts-mls"
import {getGroupMembers} from "ts-mls"
import {defaultProposalTypes} from "ts-mls"
import {processMessage} from "ts-mls"
import {unsafeTestingAuthenticationService} from "ts-mls"
import {decode} from "ts-mls"
import {mlsMessageDecoder} from "ts-mls"

// MLS Types
import {type Proposal} from "ts-mls"
import {type PrivateKeyPackage} from "ts-mls"
import {type KeyPackage} from "ts-mls"
import {type MlsContext} from "ts-mls"
import {type MlsPrivateMessage} from "ts-mls"
import {type MlsWelcomeMessage} from "ts-mls"
import {type MlsGroupInfo} from "ts-mls"
import {type CiphersuiteImpl} from "ts-mls"
import {type ClientConfig} from "ts-mls"
import {type MlsFramedMessage} from "ts-mls"

// Application Types
import {type APActor} from "../model/ap-actor"
import {type Group} from "../model/group"
import {type APKeyPackage} from "../model/ap-keypackage"
import {type Message} from "../model/message"
import {type DBKeyPackage} from "../model/db-keypackage"
import {base64ToUint8Array} from "./utils"

// IDatabase wraps all of the methods that the MLS service
// uses to store group state.
interface IDatabase {
	// load methods
	loadGroup(groupID: string): Promise<Group>
	loadMessage(messageID: string): Promise<Message>

	// save methods
	saveGroup(group: Group): Promise<void>
	saveMessage(message: Message): Promise<void>

	loadKeyPackage(): Promise<DBKeyPackage | undefined>
	saveKeyPackage(keyPackage: DBKeyPackage): Promise<void>
}

// IDelivery wraps all of the methods that the MLS service
// uses to send messages.
interface IDelivery {
	sendFramedMessage(recipients: string[], message: MlsFramedMessage): void
	sendGroupInfo(recipients: string[], message: MlsGroupInfo): void
	sendPrivateMessage(recipients: string[], message: MlsFramedMessage): void
	sendWelcome(recipients: string[], welcome: MlsWelcomeMessage): void
}

// IDirectory wraps all of the methods that the MLS service
// uses to look up users' KeyPackages.
interface IDirectory {
	getKeyPackages(actorIDs: string[]): Promise<KeyPackage[]>
	createKeyPackage(keyPackage: APKeyPackage): Promise<string>
}

interface IReceiver {
	poll(): void
}

const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"

// MLS service encrypts/decrypts messages using the MLS protocol.
// This is intended to be a reusable service that could be called
// by any software component that needs to use MLS-encrypted messages.
export class MLS {
	#database: IDatabase
	#delivery: IDelivery
	#directory: IDirectory
	#receiver: IReceiver
	#clientConfig: ClientConfig
	#cipherSuite: CiphersuiteImpl
	#publicKeyPackage: KeyPackage
	#privateKeyPackage: PrivateKeyPackage
	#actor: APActor

	constructor(
		database: IDatabase,
		delivery: IDelivery,
		directory: IDirectory,
		receiver: IReceiver,
		clientConfig: ClientConfig,

		cipherSuite: CiphersuiteImpl,
		publicKeyPackage: KeyPackage,
		privateKeyPackage: PrivateKeyPackage,
		actor: APActor,
	) {
		this.#database = database
		this.#delivery = delivery
		this.#directory = directory
		this.#receiver = receiver
		this.#clientConfig = clientConfig

		this.#actor = actor
		this.#cipherSuite = cipherSuite
		this.#publicKeyPackage = publicKeyPackage
		this.#privateKeyPackage = privateKeyPackage
	}

	/// Sending Messages

	// createGroup creates a new MLS group and saves it to the database
	async createGroup(): Promise<Group> {
		//
		const context = this.#context()
		const groupID = "uri:uuid:" + crypto.randomUUID()
		const groupIDBytes = new TextEncoder().encode(groupID)

		// Create group using ts-mls
		const clientState = await createGroup({
			context: context,
			groupId: groupIDBytes,
			keyPackage: this.#publicKeyPackage,
			privateKeyPackage: this.#privateKeyPackage,
		})

		// Populate a Group record
		const group: Group = {
			id: groupID,
			members: [],
			name: "New Group",
			clientState: clientState,
			createDate: Date.now(),
			updateDate: Date.now(),
			readDate: Date.now(),
		}

		// Save the Group
		console.log("Saving group to database:", group)
		await this.#database.saveGroup(group)

		// Success
		return group
	}

	// addGroupMembers updates the group state.  It sends a Commit
	// message to existing members, and a Welcome message to new members,
	async addGroupMembers(groupID: string, newMembers: string[]) {
		//
		const context = this.#context()

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
			context: context,
			state: group.clientState,
			extraProposals: addProposals,
			ratchetTreeExtension: true,
		})

		// Zero out the keys used to encrypt the commit message
		commitResult.consumed.forEach(zeroOutUint8Array)

		// Update the group with new state and new list of members
		group.clientState = commitResult.newState
		group.members = currentMembers.concat(newMembers)
		await this.#database.saveGroup(group)

		// Send welcome to new members
		if (commitResult.welcome != undefined) {
			this.#delivery.sendWelcome(newMembers, commitResult.welcome)
		}

		// (async) Send commit to existing members
		if (currentMembers.length > 0) {
			this.#delivery.sendFramedMessage(currentMembers, commitResult.commit)
		}
	}

	// getGroupMembers returns the list of member IDs for a given group
	async getGroupMembers(group: Group): Promise<string[]> {
		//
		// Find all current members of this group
		const leafNodes = await getGroupMembers(group.clientState)
		const members = leafNodes
			.map((leaf) => {
				const credential = leaf.credential as CredentialBasic
				if (credential.identity != undefined) {
					return new TextDecoder().decode(credential.identity)
				}
				return ""
			})
			.filter((identity) => identity != "")

		return members
	}

	async sendGroupMessage(group: string, plaintext: string): Promise<void> {
		//
		const context = this.#context()

		// Encrypt the message using the current group state
		// (This is a placeholder - the actual encryption logic will depend on the ts-mls API)
		const mlsGroup = await this.#database.loadGroup(group)

		const messageId = "uri:uuid:" + crypto.randomUUID()

		// Create the message object as JSON-LD
		const messageObject = {
			"@context": "https://www.w3.org/ns/activitystreams",
			id: messageId,
			type: "Note",
			content: plaintext,
		}

		// Encrypt the message using MLS
		const messageText = JSON.stringify(messageObject)
		const messageBytes = new TextEncoder().encode(messageText)
		const applicationMessage = await createApplicationMessage({
			context: context,
			state: mlsGroup.clientState,
			message: messageBytes,
		})

		// Zero out the keys used to encrypt the message
		applicationMessage.consumed.forEach(zeroOutUint8Array)

		// Filter out "me" from the recipients list (we don't need to send the message to ourselves)
		const recipients = mlsGroup.members.filter((member) => member !== this.#actor.id)

		// Send the message via the Delivery service
		this.#delivery.sendFramedMessage(recipients, applicationMessage.message)

		// update the Group with the new group state
		mlsGroup.clientState = applicationMessage.newState
		mlsGroup.updateDate = Date.now()
		await this.#database.saveGroup(mlsGroup)

		// Create a new Message object
		const dbMessage: Message = {
			id: messageId,
			group: group,
			sender: this.#actor.id,
			plaintext: plaintext,
			createDate: Date.now(),
		}

		// Save the message to the database
		await this.#database.saveMessage(dbMessage)
	}

	/// Receiving Messages
	// use arrow function to preserve "this" context when passing as a callback
	onMessage = async (message: string) => {
		const context = this.#context()

		console.log("MLS service: received message: ", message)
		const uintArray = base64ToUint8Array(message)
		const content = decode(mlsMessageDecoder, uintArray)!

		// Require that the we have a valid decoded message before proceeding
		if (content == undefined) {
			console.error("Unable to decode MLS message", message)
			return
		}

		console.log("Decoded message content:", content)

		switch (content.wireformat) {
			case wireformats.mls_group_info:
				console.log("Received GroupInfo message")
				return

			case wireformats.mls_key_package:
				console.log("Received KeyPackage message")
				return

			case wireformats.mls_private_message:
				this.#onMessage_PrivateMessage(content)
				return

			case wireformats.mls_public_message:
				console.log("Received PublicMessage")
				return

			case wireformats.mls_welcome:
				this.#onMessage_Welcome(content)
				return

			default:
				console.error("Unknown MLS message type:")
				return
		}
	}

	// onMessage_Welcome processes MLS "Welcome" messages that add this user to a new group.
	async #onMessage_Welcome(message: MlsWelcomeMessage) {
		console.log("Received Welcome message")
		//

		// Join the new group
		const clientState = await joinGroup({
			context: this.#context(),
			welcome: message.welcome,
			keyPackage: this.#publicKeyPackage,
			privateKeys: this.#privateKeyPackage,
		})

		// Create a new group record
		const groupId = new TextDecoder().decode(clientState.groupContext.groupId)

		const group: Group = {
			id: groupId,
			members: [],
			name: "Received Group.",
			clientState: clientState,
			createDate: Date.now(),
			updateDate: Date.now(),
			readDate: Date.now(),
		}

		// Compute members from the clientState
		group.members = await this.getGroupMembers(group)

		// Save the group to the database
		await this.#database.saveGroup(group)
	}

	// onMessage_PrivateMessage processes incoming MLS "Private Messages" that contain encrypted
	// application messages for this user.  These messages are decrypted and then processes as
	// ActivityStreams messages.
	async #onMessage_PrivateMessage(mlsMessage: MlsPrivateMessage & MlsMessageProtocol) {
		console.log("Received PrivateMessage:", mlsMessage)

		const groupId = new TextDecoder().decode(mlsMessage.privateMessage.groupId)
		const group = await this.#database.loadGroup(groupId)

		const decodedMessage = await processMessage({
			context: this.#context(),
			state: group.clientState,
			message: mlsMessage,
		})

		console.log("Processed result: ", decodedMessage)

		// Update the group state in the database
		decodedMessage.consumed.forEach(zeroOutUint8Array)
		group.clientState = decodedMessage.newState
		group.updateDate = Date.now()
		await this.#database.saveGroup(group)

		if (decodedMessage.kind != "applicationMessage") {
			console.log("Received non-application message.  Not sure what to do with these yet.")
			return
		}

		// Parse the plaintext message
		const plaintext = new TextDecoder().decode(decodedMessage.message)
		console.log("Decrypted message plaintext:", plaintext)

		const activity = JSON.parse(plaintext)
		console.log("Parsed activity:", activity)

		const message = {
			id: activity.id,
			group: groupId,
			sender: activity.actor,
			plaintext: activity.content,
			createDate: Date.now(),
		}
		console.log("Saving message to database: ", message)

		// Save the message to the database
		await this.#database.saveMessage(message)
	}

	/// Helper methods

	// Use arrow function to preserve "this" context when passing as a callback
	#context = (): MlsContext => {
		return {
			cipherSuite: this.#cipherSuite,
			authService: unsafeTestingAuthenticationService,
		}
	}
}
