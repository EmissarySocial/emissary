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
	throw new Error("Generator is empty")
}
