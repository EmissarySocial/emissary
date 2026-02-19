// rangeToArray consumes all values from a generator and returns them as an array
export function rangeToArray<T>(generator: Generator<T>): T[] {
	var result = []
	for (let value of generator) {
		result.push(value)
	}
	return result
}

// rangeFirst returns the first value from a generator
// or throws an error if the generator is empty
export function rangeFirst<T>(generator: Generator<T>): T {
	for (const value of generator) {
		return value
	}
	throw new Error("Generator is empty.")
}

// Helper to strip trailing null nodes per RFC 9420
export function stripTrailingNulls(tree: any[]): any[] {
	let lastNonNull = tree.length - 1
	while (lastNonNull >= 0 && tree[lastNonNull] === null) {
		lastNonNull--
	}
	return tree.slice(0, lastNonNull + 1)
}

// base64ToUint8Array converts a base64-encoded string to a Uint8Array
export function base64ToUint8Array(base64: string): Uint8Array {
	const binary_string = window.atob(base64)
	const len = binary_string.length
	const bytes = new Uint8Array(len)
	for (let i = 0; i < len; i++) {
		bytes[i] = binary_string.charCodeAt(i)
	}
	return bytes
}
