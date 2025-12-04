import { 
	generateKeyPackage, 
	getCiphersuiteImpl,
	getCiphersuiteFromName,

} from "ts-mls"

class KeyManager {

	constructor() {
		this.getMyKeyPackage()
	}


	private async getMyKeyPackage() {
		/*
		const impl = await getCiphersuiteImpl(getCiphersuiteFromName("MLS_256_XWING_AES256GCM_SHA512_Ed25519"))
		const aliceCredential: Credential = { credentialType: "basic", identity: new TextEncoder().encode("alice") }
		const alice = await generateKeyPackage(aliceCredential, defaultCapabilities(), defaultLifetime, [], impl)
		
		fetch("/@me/outbox")

		return alice
		*/
	}
}