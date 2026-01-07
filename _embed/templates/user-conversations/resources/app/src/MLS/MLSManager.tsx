/**
 * MLS (Message Layer Security) Manager
 * RFC 9420 implementation using ts-mls library
 * Provides end-to-end encrypted group messaging with forward secrecy
 */

import {
	createApplicationMessage,
	createCommit,
	createGroup,
	joinGroup,
	processPrivateMessage,
	processPublicMessage,
	getCiphersuiteFromName,
	generateKeyPackage,
	encodeMlsMessage,
	decodeMlsMessage,
	defaultCapabilities,
	defaultLifetime,
	emptyPskIndex,
	nobleCryptoProvider,
	type ClientState,
	type Credential,
	type Proposal,
	type PrivateKeyPackage,
	type KeyPackage,
	type Welcome,
	type PrivateMessage,
	type CiphersuiteImpl,
} from "ts-mls"

// Helper to strip trailing null nodes per RFC 9420
function stripTrailingNulls(tree: any[]): any[] {
	let lastNonNull = tree.length - 1
	while (lastNonNull >= 0 && tree[lastNonNull] === null) {
		lastNonNull--
	}
	return tree.slice(0, lastNonNull + 1)
}

export interface MLSGroupInfo {
	groupId: Uint8Array
	members: string[]
	epoch: bigint
}

export interface MLSMessageEnvelope {
	groupId: Uint8Array
	ciphertext: Uint8Array
	timestamp: number
}

export interface MLSKeyPackageBundle {
	publicPackage: KeyPackage
	privatePackage?: PrivateKeyPackage
	userId: string
}

/**
 * MLSManager wraps the ts-mls functional API with a class-based interface
 * for easier state management in applications
 */
export class MLSManager {
	private userId: string
	private cipherSuite: CiphersuiteImpl | null = null
	private initialized: boolean = false
	private groups: Map<string, ClientState> = new Map()
	private keyPackage: MLSKeyPackageBundle | null = null
	private credential: Credential

	constructor(userId: string) {
		this.userId = userId
		this.credential = {
			credentialType: "basic",
			identity: new TextEncoder().encode(userId),
		}
	}

	/**
	 * Initialize the MLS client with a ciphersuite
	 */
	async initialize(): Promise<void> {
		if (this.initialized) {
			return
		}

		try {
			// Use MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519 (ID: 1)
			// Using nobleCryptoProvider for compatibility (pure JS implementation)
			const cipherSuiteName = "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
			const cs = getCiphersuiteFromName(cipherSuiteName)
			this.cipherSuite = await nobleCryptoProvider.getCiphersuiteImpl(cs)

			// Generate initial key package for this user
			await this.generateKeyPackage()

			// Mark as initialized after successful setup
			this.initialized = true
		} catch (error) {
			throw error
		}
	}

	/**
	 * Generate a new key package for joining groups
	 */
	async generateKeyPackage(): Promise<MLSKeyPackageBundle> {
		try {
			const keyPackageResult = await generateKeyPackage(
				this.credential,
				defaultCapabilities(),
				defaultLifetime,
				[],
				this.cipherSuite!
			)

			this.keyPackage = {
				...keyPackageResult,
				userId: this.userId,
			}

			return this.keyPackage
		} catch (error) {
			throw error
		}
	}

	/**
	 * Get the current key package
	 */
	getKeyPackage(): MLSKeyPackageBundle | null {
		return this.keyPackage
	}

	/**
	 * Create a new MLS group
	 */
	async createGroup(groupId: string): Promise<MLSGroupInfo> {
		this.ensureInitialized()

		if (!this.keyPackage) {
			throw new Error("No key package available. Call generateKeyPackage() first.")
		}

		if (this.keyPackage.privatePackage == undefined) {
			throw new Error("No private key package available.")
		}

		const groupIdBytes = new TextEncoder().encode(groupId)

		// Create group using ts-mls
		const groupState = await createGroup(
			groupIdBytes,
			this.keyPackage.publicPackage,
			this.keyPackage.privatePackage,
			[],
			this.cipherSuite!
		)

		this.groups.set(groupId, groupState)

		const groupInfo: MLSGroupInfo = {
			groupId: groupIdBytes,
			members: [this.userId],
			epoch: groupState.groupContext.epoch,
		}

		return groupInfo
	}

	/**
	 * Add members to an existing group
	 */
	async addMembers(
		groupId: string,
		keyPackages: MLSKeyPackageBundle[]
	): Promise<{welcome: Welcome; ratchetTree: any; commit: any}> {
		this.ensureInitialized()

		const groupState = this.groups.get(groupId)
		if (!groupState) {
			throw new Error(`Group ${groupId} not found`)
		}

		// Create add proposals for each key package
		const addProposals: Proposal[] = keyPackages.map((kp) => ({
			proposalType: "add",
			add: {
				keyPackage: kp.publicPackage,
			},
		}))

		// Create commit with add proposals
		const commitResult = await createCommit(
			{state: groupState, cipherSuite: this.cipherSuite!},
			{extraProposals: addProposals}
		)

		// Update group state
		this.groups.set(groupId, commitResult.newState)

		if (!commitResult.welcome) {
			throw new Error("No welcome message generated")
		}

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
	}

	/**
	 * Process a Welcome message to join an MLS group
	 *
	 * RFC 9420 Compliance:
	 * - Interior null nodes represent blank parent nodes (unmerged positions)
	 * - These nulls are REQUIRED for proper binary tree structure
	 * - Trailing nulls are stripped by sender (per RFC 9420 requirement)
	 * - ratchetTree parameter is optional; ts-mls can extract from Welcome extension
	 *
	 * @param welcome - The Welcome message from group creator
	 * @param ratchetTree - Optional ratchet tree (normally provided out-of-band)
	 */
	async processWelcome(welcome: Welcome, ratchetTree?: Uint8Array[]): Promise<MLSGroupInfo> {
		this.ensureInitialized()

		if (!this.keyPackage) {
			throw new Error("No key package available")
		}

		if (!this.keyPackage.privatePackage) {
			throw new Error("No private key package available")
		}

		// RFC 9420: Interior null nodes are valid (represent blank parent nodes)
		// Trailing nulls are stripped by sender per RFC requirement
		// Simply pass the tree as-is to ts-mls joinGroup()

		if (ratchetTree && Array.isArray(ratchetTree)) {
			const nullCount = ratchetTree.filter((n) => n === null).length
			// DEBUG: Log structure of each node
			console.group("🔍 [MLS Debug] Ratchet Tree Structure")
			ratchetTree.forEach((node, i) => {
				if (node !== null) {
					console.log({
						index: i,
						isObject: typeof node === "object",
						hasNodeType: node && "nodeType" in node,
						nodeType: node?.nodeType,
						keys: node && typeof node === "object" ? Object.keys(node).slice(0, 5) : "n/a",
					})
				}
			})
			console.groupEnd()
		}

		const groupState = await joinGroup(
			welcome,
			this.keyPackage.publicPackage,
			this.keyPackage.privatePackage,
			emptyPskIndex,
			this.cipherSuite!,
			ratchetTree // Pass as-is - nulls are valid
		)

		const groupId = new TextDecoder().decode(groupState.groupContext.groupId)
		this.groups.set(groupId, groupState)

		// Extract member identities from ratchet tree
		const members = this.extractMembersFromState(groupState)

		const groupInfo: MLSGroupInfo = {
			groupId: groupState.groupContext.groupId,
			members,
			epoch: groupState.groupContext.epoch,
		}

		return groupInfo
	}

	/**
	 * Encrypt a message for a group
	 */
	async encryptMessage(groupId: string, plaintext: string): Promise<MLSMessageEnvelope> {
		this.ensureInitialized()

		try {
			const groupState = this.groups.get(groupId)
			if (!groupState) {
				throw new Error(`Group ${groupId} not found`)
			}

			const plaintextBytes = new TextEncoder().encode(plaintext)

			// Create application message
			const result = await createApplicationMessage(groupState, plaintextBytes, this.cipherSuite!)

			// Update group state (for key ratcheting)
			this.groups.set(groupId, result.newState)

			// Encode the private message
			const encoded = encodeMlsMessage({
				privateMessage: result.privateMessage,
				wireformat: "mls_private_message",
				version: "mls10",
			})

			const envelope: MLSMessageEnvelope = {
				groupId: new TextEncoder().encode(groupId),
				ciphertext: encoded,
				timestamp: Date.now(),
			}

			return envelope
		} catch (error) {
			throw error
		}
	}

	/**
	 * Decrypt a message from a group
	 */
	async decryptMessage(envelope: MLSMessageEnvelope): Promise<string> {
		this.ensureInitialized()

		try {
			const groupId = new TextDecoder().decode(envelope.groupId)
			const groupState = this.groups.get(groupId)
			if (!groupState) {
				throw new Error(`Group ${groupId} not found`)
			}

			// Decode the message
			const decoded = decodeMlsMessage(envelope.ciphertext, 0)
			if (!decoded) {
				throw new Error("Failed to decode message")
			}

			const [decodedMessage] = decoded
			if (decodedMessage.wireformat !== "mls_private_message") {
				throw new Error("Expected private message")
			}

			// Process the private message
			const result = await processPrivateMessage(
				groupState,
				decodedMessage.privateMessage,
				emptyPskIndex,
				this.cipherSuite!
			)

			// Update group state
			this.groups.set(groupId, result.newState)

			if (result.kind !== "applicationMessage") {
				throw new Error("Expected application message")
			}

			const plaintext = new TextDecoder().decode(result.message)

			return plaintext
		} catch (error) {
			throw error
		}
	}

	/**
	 * Update the group keys (key rotation)
	 */
	async updateKey(groupId: string): Promise<any> {
		this.ensureInitialized()

		try {
			const groupState = this.groups.get(groupId)
			if (!groupState) {
				throw new Error(`Group ${groupId} not found`)
			}

			// Create update commit (forces path update)
			const commitResult = await createCommit(
				{state: groupState, cipherSuite: this.cipherSuite!}
				// {forcePathUpdate: true} removed because this option doesn't exist in ts-mls
			)

			// Update group state
			this.groups.set(groupId, commitResult.newState)

			// Return the raw commit object for other members to process
			return commitResult.commit
		} catch (error) {}
	}

	/**
	 * Process a commit message (key rotation, member changes)
	 *
	 * RFC 9420 Section 12.1.8:
	 * - Update commits (key rotation) → PrivateMessage
	 * - Add/Remove commits → PublicMessage (for existing group members)
	 *
	 * This implementation handles both types based on wireformat.
	 */
	async processCommit(groupId: string, commit: any): Promise<void> {
		this.ensureInitialized()

		try {
			// DETAILED DEBUG LOGGING
			console.group("🔍 [MLS Debug] Full Commit Structure")

			// Log proposals if present
			if (commit.publicMessage?.content) {
				if (commit.publicMessage.content.proposals) {
					commit.publicMessage.content.proposals.forEach((prop: any, i: number) => {
						console.log({
							index: i,
							keys: Object.keys(prop),
							full: prop,
						})
					})
				}
			}
			console.groupEnd()

			const groupState = this.groups.get(groupId)
			if (!groupState) {
				throw new Error(`Group ${groupId} not found`)
			}

			let result

			// RFC 9420: Route based on message type
			if (commit.wireformat === "mls_public_message") {
				// Public messages (add/remove member commits)

				result = await processPublicMessage(groupState, commit.publicMessage, emptyPskIndex, this.cipherSuite!)
			} else if (commit.wireformat === "mls_private_message") {
				// Private messages (update/key rotation commits)

				result = await processPrivateMessage(
					groupState,
					commit.privateMessage,
					emptyPskIndex,
					this.cipherSuite!
				)
			} else {
				throw new Error(`Unknown commit wireformat: ${commit.wireformat}`)
			}

			// Update group state
			this.groups.set(groupId, result.newState)
		} catch (error) {
			console.error("Error processing commit:", error)
			throw error
		}
	}

	/**
	 * Remove members from a group
	 */
	async removeMembers(groupId: string, memberIndices: number[]): Promise<Uint8Array> {
		this.ensureInitialized()

		const groupState = this.groups.get(groupId)
		if (!groupState) {
			throw new Error(`Group ${groupId} not found`)
		}

		// Create remove proposals
		const removeProposals: Proposal[] = memberIndices.map((index) => ({
			proposalType: "remove",
			remove: {
				removed: BigInt(index),
			},
		}))

		// Create commit with remove proposals
		const commitResult = await createCommit(
			{state: groupState, cipherSuite: this.cipherSuite!},
			{extraProposals: removeProposals}
		)

		// Update group state
		this.groups.set(groupId, commitResult.newState)

		// Encode the commit
		const encodedCommit = encodeMlsMessage({
			publicMessage: commitResult.publicMessage!,
			wireformat: "mls_public_message",
			version: "mls10",
		})
		return encodedCommit
	}

	/**
	 * Get list of groups
	 */
	async getGroups(): Promise<Uint8Array[]> {
		this.ensureInitialized()
		return Array.from(this.groups.keys()).map((id) => new TextEncoder().encode(id))
	}

	/**
	 * Export group state for persistence
	 */
	async exportGroupState(groupId: string): Promise<any> {
		this.ensureInitialized()

		try {
			const groupState = this.groups.get(groupId)
			if (!groupState) {
				throw new Error(`Group ${groupId} not found`)
			}

			// Note: ts-mls ClientState contains non-serializable crypto keys
			// This is a simplified export - in production you'd need proper serialization
			const exportData = {
				groupId,
				epoch: groupState.groupContext.epoch.toString(),
				exported: Date.now(),
				// Add other serializable fields as needed
			}
		} catch (error) {}
	}

	/**
	 * Get user ID
	 */
	getUserId(): string {
		return this.userId
	}

	/**
	 * Get group information
	 */
	async getGroupKeyInfo(groupId: string): Promise<any> {
		const groupState = this.groups.get(groupId)

		if (!groupState) {
			return null
		}

		const members = this.extractMembersFromState(groupState)

		return {
			groupId,
			epoch: groupState.groupContext.epoch.toString(),
			members,
			cipherSuite: "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519",
			treeHash: this.bytesToHex(groupState.groupContext.treeHash).substring(0, 16),
		}
	}

	/**
	 * Clean up resources
	 */
	async destroy(): Promise<void> {
		this.keyPackage = null
		this.initialized = false
	}

	/**
	 * Extract member identities from group state
	 */
	private extractMembersFromState(state: ClientState): string[] {
		const members: string[] = []

		// Iterate through ratchet tree to find leaf nodes
		for (let i = 0; i < state.ratchetTree.length; i++) {
			const node = state.ratchetTree[i]
			if (node && node.nodeType === "leaf" && node.leaf.credential) {
				const identity = new TextDecoder().decode(node.leaf.credential.identity)
				members.push(identity)
			}
		}

		return members
	}

	/**
	 * Convert bytes to hex string
	 */
	private bytesToHex(bytes: Uint8Array): string {
		return Array.from(bytes)
			.map((b) => b.toString(16).padStart(2, "0"))
			.join("")
	}

	/**
	 * Ensure the manager is initialized
	 */
	private ensureInitialized(): void {
		if (!this.initialized) {
			throw new Error("MLSManager not initialized. Call initialize() first.")
		}
	}
}

export default MLSManager
