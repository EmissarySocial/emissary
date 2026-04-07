package emojikey

import (
	"crypto/sha256"
	"encoding/binary"
)

// EmojiKey generates a human-friendly representation of a public key using emojis.
// It returns both the raw checksum string and the corresponding emoji sequence.
func EmojiKey(publicKey []byte) (string, [5]Emoji) {

	// Calculate checksum and collect emoji values
	checksum := KeyChecksum(publicKey)
	checksumString := string(checksum[:])
	emojiKey := EmojiValues(checksum)

	// Station.
	return checksumString, emojiKey
}

// EmojiValues returns a sequence of 5 emojis derived from the provided checksum
func EmojiValues(checksum [32]byte) [5]Emoji {

	// Collect the emoji set and prepare a result array
	emojis := Emojis()
	var result [5]Emoji

	// Extract 5 non-overlapping 2-byte windows from the hash → 5 indices
	for index := range 5 {

		// Get a 2-byte window from the checksum
		first := checksum[index*2]
		second := checksum[index*2+1]
		numBytes := []byte{first, second}

		// Convert the 2-byte window to a big-endian uint16 modulo (number of emojis)
		intValue := binary.BigEndian.Uint16(numBytes)
		intValue = intValue % uint16(len(emojis))

		// Pick the emoji at the corresponding index
		result[index] = emojis[intValue]
	}

	// Return the 5 selected indexes
	return result
}

// KeyChecksum generates a SHA-256 checksum of the provided PublicKey bytes.
func KeyChecksum(publicKey []byte) [32]byte {
	return sha256.Sum256(publicKey)
}

/*
	hash = SHA-256(signature_key_bytes)              // 32 bytes
	for i in 0..5:
	index = big_endian_u16(hash[i*2], hash[i*2+1]) // 2-byte window
	emojis[i] = EMOJI_SET[index % 350]
	return emojis

	// Index extraction: Take 5 non-overlapping 2-byte (big-endian u16) windows from the hash → 5 indices


	// Emoji lookup: Each index modulo the emoji table size → one emoji per index
	// Output: An ordered sequence of 5 emoji + description (per client)

}



	// sigKey: Uint8Array (raw key bytes) or hex string
async function signatureKeyToEmojiFingerprint(sigKey, emojiSet) {
	const bytes = typeof sigKey === 'string'
	? Uint8Array.from(sigKey.match(/.{2}/g), h => parseInt(h, 16))
	: sigKey;
	const hash = new Uint8Array(await crypto.subtle.digest('SHA-256', bytes));
	return Array.from({ length: 5 }, (_, i) => {
	const idx = ((hash[i * 2] << 8) | hash[i * 2 + 1]) % 350;
	return emojiSet[idx];
	});
}
}*/
