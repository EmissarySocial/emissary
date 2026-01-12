export type Config = {
	id: string
	ready: boolean // TRUE when an actual value has been loaded from the database
	welcome: boolean // TRUE when the user has passed the initial welcome screen
	hasEncryptionKeys: boolean // TRUE when the user has created encryption keys
	password: string // TODO: TEMPORARY: TO BE REMOVED.
	passwordHint: string // Hint to help the user remember their password
	clientName: string // Name of this client/device
}

export const ConfigID = "config"

export function NewConfig(): Config {
	return {
		id: ConfigID,
		ready: false,
		welcome: false,
		hasEncryptionKeys: false,
		password: "",
		passwordHint: "",
		clientName: "Unknown Device",
	}
}
