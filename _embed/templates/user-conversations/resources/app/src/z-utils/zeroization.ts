/**
 * Security utility for zeroizing sensitive data from memory
 *
 * Note: JavaScript doesn't guarantee memory clearing, but this helps
 * reduce exposure window and follows security best practices.
 *
 * Best practices:
 * - Always zeroize sensitive buffers after use
 * - Zeroize in finally blocks to ensure cleanup even on exceptions
 * - Zeroize before throwing errors to prevent leaks
 * - Clear references to CryptoKey objects (set to null)
 */

/**
 * Zeroization utility class for clearing sensitive data from memory
 */
export class Zeroization {
  /**
   * Zeroize a Uint8Array buffer by filling it with zeros
   *
   * @param buffer - The buffer to zeroize (can be null or undefined)
   */
  static zeroize(buffer: Uint8Array | null | undefined): void {
    if (buffer && buffer.length > 0) {
      buffer.fill(0);
    }
  }

  /**
   * Zeroize an ArrayBuffer by filling it with zeros
   *
   * @param buffer - The ArrayBuffer to zeroize (can be null or undefined)
   */
  static zeroizeArrayBuffer(buffer: ArrayBuffer | null | undefined): void {
    if (buffer && buffer.byteLength > 0) {
      new Uint8Array(buffer).fill(0);
    }
  }

  /**
   * Zeroize a string by clearing its reference
   *
   * Note: Strings are immutable in JavaScript, so we can't actually
   * clear the memory. This is a best-effort approach that at least
   * ensures we don't keep references to sensitive strings.
   *
   * @param str - The string to "zeroize" (can be null or undefined)
   */
  static zeroizeString(str: string | null | undefined): void {
    // Strings are immutable in JS, so we can't actually clear the memory
    // The best we can do is ensure we don't keep references
    // The actual memory will be garbage collected
    // This method exists for API consistency
    if (str) {
      // No-op: strings are immutable, but we document the intent
    }
  }

  /**
   * Zeroize multiple buffers at once
   *
   * @param buffers - Variable number of buffers to zeroize
   */
  static zeroizeAll(
    ...buffers: (Uint8Array | ArrayBuffer | null | undefined)[]
  ): void {
    for (const buf of buffers) {
      if (buf instanceof Uint8Array) {
        this.zeroize(buf);
      } else if (buf instanceof ArrayBuffer) {
        this.zeroizeArrayBuffer(buf);
      }
      // null/undefined are handled gracefully (no-op)
    }
  }
}
