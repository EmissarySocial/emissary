import { type KeyPackage } from "ts-mls"
import { type PrivateKeyPackage } from "ts-mls"

// Record stored in the local IndexedDB database
export type IDBMLSKeyPackage = {
	id: string // The URL of this KeyPackage
	publicPackage: KeyPackage
	privatePackage: PrivateKeyPackage
}
