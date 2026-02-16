import {
	createApplicationMessage,
	createCommit,
	createGroup,
	defaultProposalTypes,
	defaultCredentialTypes,
	joinGroup,
	processMessage,
	getCiphersuiteImpl,
	type Credential,
	defaultCapabilities,
	defaultLifetime,
	generateKeyPackage,
	type MlsContext,
	encode,
	decode,
	mlsMessageEncoder,
	mlsMessageDecoder,
	protocolVersions,
	unsafeTestingAuthenticationService,
	wireformats,
	type Proposal,
	zeroOutUint8Array,
	ciphersuites,
	getCiphersuiteFromName,
} from "ts-mls"

// Copy/paste from https://github.com/LukaJCB/ts-mls
;(async function demo(): Promise<void> {
	const cs = getCiphersuiteFromName("MLS_256_XWING_AES256GCM_SHA512_Ed25519")
	const impl = await getCiphersuiteImpl(cs)

	const context: MlsContext = {
		cipherSuite: impl,
		authService: unsafeTestingAuthenticationService,
	}

	// alice generates her key package
	const aliceCredential: Credential = {
		credentialType: defaultCredentialTypes.basic,
		identity: new TextEncoder().encode("alice"),
	}
	const alice = await generateKeyPackage({credential: aliceCredential, cipherSuite: impl})

	const groupId = new TextEncoder().encode("group1")

	// alice creates a new group
	let aliceGroup = await createGroup({
		context,
		groupId,
		keyPackage: alice.publicPackage,
		privateKeyPackage: alice.privatePackage,
	})

	// bob generates his key package
	const bobCredential: Credential = {
		credentialType: defaultCredentialTypes.basic,
		identity: new TextEncoder().encode("bob"),
	}
	const bob = await generateKeyPackage({credential: bobCredential, cipherSuite: impl})

	// bob sends keyPackage to alice
	const keyPackageMessage = encode(mlsMessageEncoder, {
		keyPackage: bob.publicPackage,
		wireformat: wireformats.mls_key_package,
		version: protocolVersions.mls10,
	})

	// alice decodes bob's keyPackage
	const decodedKeyPackage = decode(mlsMessageDecoder, keyPackageMessage)!

	if (decodedKeyPackage.wireformat !== wireformats.mls_key_package) throw new Error("Expected key package")

	// alice creates proposal to add bob
	const addBobProposal: Proposal = {
		proposalType: defaultProposalTypes.add,
		add: {
			keyPackage: decodedKeyPackage.keyPackage,
		},
	}

	// alice commits
	const commitResult = await createCommit({
		context,
		state: aliceGroup,
		extraProposals: [addBobProposal],
	})

	aliceGroup = commitResult.newState

	// alice deletes the keys used to encrypt the commit message
	commitResult.consumed.forEach(zeroOutUint8Array)

	// alice sends welcome message to bob
	const encodedWelcome = encode(mlsMessageEncoder, commitResult.welcome!)

	// bob decodes the welcome message
	const decodedWelcome = decode(mlsMessageDecoder, encodedWelcome)!

	if (decodedWelcome.wireformat !== wireformats.mls_welcome) throw new Error("Expected welcome")

	// bob creates his own group state
	let bobGroup = await joinGroup({
		context,
		welcome: decodedWelcome.welcome,
		keyPackage: bob.publicPackage,
		privateKeys: bob.privatePackage,
		ratchetTree: aliceGroup.ratchetTree,
	})

	const messageToBob = new TextEncoder().encode("Hello bob!")

	// alice creates a message to the group
	const aliceCreateMessageResult = await createApplicationMessage({
		context,
		state: aliceGroup,
		message: messageToBob,
	})

	aliceGroup = aliceCreateMessageResult.newState

	// alice deletes the keys used to encrypt the application message
	aliceCreateMessageResult.consumed.forEach(zeroOutUint8Array)

	// alice sends the message to bob
	const encodedPrivateMessageAlice = encode(mlsMessageEncoder, aliceCreateMessageResult.message)

	// bob decodes the message
	const decodedPrivateMessageAlice = decode(mlsMessageDecoder, encodedPrivateMessageAlice)!

	if (decodedPrivateMessageAlice.wireformat !== wireformats.mls_private_message)
		throw new Error("Expected private message")

	// bob receives the message
	const bobProcessMessageResult = await processMessage({
		context,
		state: bobGroup,
		message: decodedPrivateMessageAlice,
	})

	bobGroup = bobProcessMessageResult.newState

	if (bobProcessMessageResult.kind === "newState") throw new Error("Expected application message")

	// bob deletes the keys used to decrypt the application message
	bobProcessMessageResult.consumed.forEach(zeroOutUint8Array)

	console.log(bobProcessMessageResult.message)
})()
