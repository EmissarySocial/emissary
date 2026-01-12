import type {CiphersuiteImpl} from "ts-mls"
import type {KeyPackage} from "ts-mls"
import type {PrivateKeyPackage} from "ts-mls"

export type DBKeyPackage = {
	id: string
	keyPackageURL: string
	clientName: string
	publicKeyPackage: KeyPackage
	privateKeyPackage: PrivateKeyPackage
	cipherSuiteName: "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
}
