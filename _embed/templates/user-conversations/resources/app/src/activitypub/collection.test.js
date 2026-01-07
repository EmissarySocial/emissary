import {describe, it, expect} from "vitest"
import {Collection} from "./collection"

describe("Collection", () => {
	describe("constructor", () => {
		it("should create an empty collection", () => {
			const collection = new Collection()
			expect(collection.totalItems).toBe(0)
			expect(collection.items).toEqual([])
		})

		it("should initialize from ActivityPub collection object", () => {
			const data = {
				type: "Collection",
				totalItems: 3,
				items: ["item1", "item2", "item3"],
			}
			const collection = new Collection(data)
			expect(collection.totalItems).toBe(3)
			expect(collection.items).toEqual(["item1", "item2", "item3"])
		})

		it("should handle OrderedCollection type", () => {
			const data = {
				type: "OrderedCollection",
				totalItems: 2,
				orderedItems: ["first", "second"],
			}
			const collection = new Collection(data)
			expect(collection.totalItems).toBe(2)
			expect(collection.items).toEqual(["first", "second"])
		})

		it("should handle missing totalItems", () => {
			const data = {
				type: "Collection",
				items: ["item1"],
			}
			const collection = new Collection(data)
			expect(collection.totalItems).toBe(0)
			expect(collection.items).toEqual(["item1"])
		})

		it("should handle missing items array", () => {
			const data = {
				type: "Collection",
				totalItems: 5,
			}
			const collection = new Collection(data)
			expect(collection.totalItems).toBe(5)
			expect(collection.items).toEqual([])
		})
	})

	describe("first/last/next/prev properties", () => {
		it("should extract first page reference", () => {
			const collection = new Collection({
				first: "https://example.com/page1",
			})
			expect(collection.first).toBe("https://example.com/page1")
		})

		it("should extract first page from object", () => {
			const collection = new Collection({
				first: {id: "https://example.com/page1", type: "CollectionPage"},
			})
			expect(collection.first).toBe("https://example.com/page1")
		})

		it("should extract last page reference", () => {
			const collection = new Collection({
				last: "https://example.com/page-last",
			})
			expect(collection.last).toBe("https://example.com/page-last")
		})

		it("should extract next page reference", () => {
			const collection = new Collection({
				next: "https://example.com/page2",
			})
			expect(collection.next).toBe("https://example.com/page2")
		})

		it("should extract prev page reference", () => {
			const collection = new Collection({
				prev: "https://example.com/page0",
			})
			expect(collection.prev).toBe("https://example.com/page0")
		})

		it("should return undefined for missing pagination links", () => {
			const collection = new Collection()
			expect(collection.first).toBeUndefined()
			expect(collection.last).toBeUndefined()
			expect(collection.next).toBeUndefined()
			expect(collection.prev).toBeUndefined()
		})
	})

	describe("isEmpty", () => {
		it("should return true for empty collection", () => {
			const collection = new Collection()
			expect(collection.isEmpty()).toBe(true)
		})

		it("should return false when items exist", () => {
			const collection = new Collection({
				items: ["item1"],
			})
			expect(collection.isEmpty()).toBe(false)
		})

		it("should return true when items is empty array", () => {
			const collection = new Collection({
				items: [],
			})
			expect(collection.isEmpty()).toBe(true)
		})
	})

	describe("hasNext/hasPrev", () => {
		it("should return true when next exists", () => {
			const collection = new Collection({
				next: "https://example.com/page2",
			})
			expect(collection.hasNext()).toBe(true)
		})

		it("should return false when next is missing", () => {
			const collection = new Collection()
			expect(collection.hasNext()).toBe(false)
		})

		it("should return true when prev exists", () => {
			const collection = new Collection({
				prev: "https://example.com/page0",
			})
			expect(collection.hasPrev()).toBe(true)
		})

		it("should return false when prev is missing", () => {
			const collection = new Collection()
			expect(collection.hasPrev()).toBe(false)
		})
	})

	describe("toJSON", () => {
		it("should serialize collection to JSON", () => {
			const collection = new Collection({
				type: "Collection",
				totalItems: 2,
				items: ["a", "b"],
				first: "https://example.com/first",
			})
			const json = collection.toJSON()
			expect(json.type).toBe("Collection")
			expect(json.totalItems).toBe(2)
			expect(json.items).toEqual(["a", "b"])
			expect(json.first).toBe("https://example.com/first")
		})

		it("should omit undefined properties", () => {
			const collection = new Collection({
				totalItems: 1,
				items: ["item"],
			})
			const json = collection.toJSON()
			expect(json).not.toHaveProperty("next")
			expect(json).not.toHaveProperty("prev")
			expect(json).not.toHaveProperty("first")
			expect(json).not.toHaveProperty("last")
		})
	})
})
