import type {KeyPackage, PrivateKeyPackage} from "ts-mls"

export type DBKeyPackage = {
	keyPackageID: string
	publicKeyPackage: KeyPackage
	privateKeyPackage: PrivateKeyPackage
}
