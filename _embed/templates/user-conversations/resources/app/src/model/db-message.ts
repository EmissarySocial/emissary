export type DBMessage = {
	messageID: string
	groupID: string
	senderID: string
	ciphertext: Uint8Array
	plaintext: string
	createDate: number
}
