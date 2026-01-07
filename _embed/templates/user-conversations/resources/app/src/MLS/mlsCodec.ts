/**
 * MLS Message encoding/decoding utilities
 *
 * Since ts-mls doesn't export encode/decode functions from the main package,
 * we'll use JSON serialization with Uint8Array and BigInt conversion for transmission.
 * This works for KeyPackage, Welcome, Commit, and other MLS messages.
 */

/**
 * Convert Uint8Array and BigInt to regular array/string recursively
 */
function uint8ArrayToArray(obj: any): any {
	// Handle null explicitly (important for MLS ratchet trees)
	if (obj === null) {
		return null
	}
	if (obj instanceof Uint8Array) {
		return {__type: "Uint8Array", data: Array.from(obj)}
	}
	if (typeof obj === "bigint") {
		return {__type: "BigInt", value: obj.toString()}
	}
	if (Array.isArray(obj)) {
		return obj.map(uint8ArrayToArray)
	}
	if (typeof obj === "object") {
		const result: any = {}
		for (const key in obj) {
			if (obj.hasOwnProperty(key)) {
				result[key] = uint8ArrayToArray(obj[key])
			}
		}
		return result
	}
	return obj
}

/**
 * Convert regular arrays back to Uint8Array and strings back to BigInt recursively
 */
function arrayToUint8Array(obj: any): any {
	// Handle null: Convert to undefined for MLS compatibility
	// ts-mls expects RatchetTree = (Node | undefined)[], not (Node | null)[]
	if (obj === null) {
		return undefined
	}

	// Check for marked Uint8Array
	if (typeof obj === "object" && obj.__type === "Uint8Array") {
		return new Uint8Array(obj.data)
	}

	// Check for marked BigInt
	if (typeof obj === "object" && obj.__type === "BigInt") {
		return BigInt(obj.value)
	}

	if (Array.isArray(obj)) {
		return obj.map(arrayToUint8Array)
	}

	if (typeof obj === "object") {
		// Check if this is an array-like object with sequential numeric keys
		const keys = Object.keys(obj)
		const isArrayLike = keys.length > 0 && keys.every((key, index) => key === String(index))

		if (isArrayLike) {
			// Convert object with numeric keys back to array
			const result: any[] = []
			for (const key in obj) {
				if (obj.hasOwnProperty(key)) {
					result[parseInt(key)] = arrayToUint8Array(obj[key])
				}
			}
			return result
		}

		const result: any = {}
		for (const key in obj) {
			if (obj.hasOwnProperty(key)) {
				result[key] = arrayToUint8Array(obj[key])
			}
		}
		return result
	}

	return obj
}

/**
 * Encode a KeyPackage to JSON string for transmission
 */
export function encodeKeyPackage(keyPackage: any): string {
	// Convert Uint8Arrays to regular arrays so they can be JSON serialized
	const serializable = uint8ArrayToArray(keyPackage)
	return JSON.stringify(serializable)
}

/**
 * Decode JSON string back to a KeyPackage object
 */
export function decodeKeyPackage(encoded: string): any {
	const parsed = JSON.parse(encoded)
	// Convert arrays back to Uint8Arrays
	return arrayToUint8Array(parsed)
}

/**
 * Encode a Welcome message to JSON string for transmission
 */
export function encodeWelcome(welcome: any): string {
	const serializable = uint8ArrayToArray(welcome)
	return JSON.stringify(serializable)
}

/**
 * Decode JSON string back to a Welcome object
 */
export function decodeWelcome(encoded: string): any {
	const parsed = JSON.parse(encoded)
	return arrayToUint8Array(parsed)
}

/**
 * Encode a Commit message to JSON string for transmission
 */
export function encodeCommit(commit: any): string {
	const serializable = uint8ArrayToArray(commit)
	return JSON.stringify(serializable)
}

/**
 * Decode JSON string back to a Commit object
 */
export function decodeCommit(encoded: string): any {
	const parsed = JSON.parse(encoded)
	return arrayToUint8Array(parsed)
}

/**
 * Encode a RatchetTree to JSON string for transmission
 */
export function encodeRatchetTree(ratchetTree: any): string {
	const serializable = uint8ArrayToArray(ratchetTree)
	return JSON.stringify(serializable)
}

/**
 * Decode JSON string back to a RatchetTree
 */
export function decodeRatchetTree(encoded: string): any {
	const parsed = JSON.parse(encoded)

	if (parsed[0] && typeof parsed[0] === "object") {
		console.log("Node 0 is an object")
	}
	if (parsed[2] && typeof parsed[2] === "object") {
		console.log("Node 2 is an object")
	}

	const result = arrayToUint8Array(parsed)

	if (result[0] && typeof result[0] === "object") {
		console.log("Result[0] is an object")
	}
	if (result[2] && typeof result[2] === "object") {
		console.log("Result[2] is an object")
	}
	return result
}
