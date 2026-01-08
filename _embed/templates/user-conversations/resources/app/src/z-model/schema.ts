import { type DBSchema } from "idb"
import type { IDBConversation } from "./conversation"
import type { IDBMessage } from "./message"
import type { IDBMLSKeyPackage } from "./mlsKeyPackage"
import type { IDBMLSMessage } from "./mlsMessage"
import type { IDBMLSGroup } from "./mlsGroup"

interface Schema extends DBSchema {

	conversation: {
		key: string,
		value: IDBConversation,
	}

	message: {
		key: string,
		value: IDBMessage,
	}

	mlsGroup: {
		key: string,
		value: IDBMLSGroup,
	}

	mlsKeyPackages: {
		key: string,
		value: IDBMLSKeyPackage,
	},

	mlsMessages: {
		key: string,
		value: IDBMLSMessage,
	}
}
