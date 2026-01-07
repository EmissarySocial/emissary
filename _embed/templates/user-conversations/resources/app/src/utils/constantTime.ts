/**
 * Constant-Time Comparison Utility
 *
 * Provides constant-time comparison functions to prevent timing-based
 * side-channel attacks in security-critical operations.
 *
 * Note: True constant-time execution is difficult to achieve in JavaScript
 * due to JIT compilation, branch prediction, and other runtime optimizations.
 * This implementation provides best-effort constant-time guarantees by:
 * - Always comparing all bytes/characters regardless of early differences
 * - Using bitwise XOR operations to accumulate differences
 * - Avoiding short-circuiting and early returns
 * - Performing full comparisons even when mismatches are detected early
 *
 * While perfect constant-time is impossible in JavaScript, this reduces
 * timing variance significantly compared to regular string/buffer comparisons.
 */

/**
 * Constant-time comparison utility class
 */
export class ConstantTime {
  /**
   * Constant-time string comparison
   *
   * Compares two strings in constant time to prevent timing attacks.
   * Always compares all characters regardless of early differences.
   *
   * @param str1 - First string to compare
   * @param str2 - Second string to compare
   * @returns True if strings are equal, false otherwise
   * @throws Error if inputs are null, undefined, or not strings
   */
  static constantTimeCompareStrings(str1: string, str2: string): boolean {
    // Validate inputs
    if (str1 === null || str1 === undefined || typeof str1 !== "string") {
      throw new Error("First argument must be a string");
    }
    if (str2 === null || str2 === undefined || typeof str2 !== "string") {
      throw new Error("Second argument must be a string");
    }

    const len1 = str1.length;
    const len2 = str2.length;

    // Track length mismatch (but continue comparison)
    let lengthMismatch = len1 !== len2 ? 1 : 0;

    // Always process the maximum length to ensure constant-time behavior
    // This prevents timing leaks based on string length differences
    let diff = 0;
    const maxLen = Math.max(len1, len2);

    for (let i = 0; i < maxLen; i++) {
      // Get character codes, using 0 for out-of-bounds access
      // This ensures we always process the same number of iterations
      const c1 = i < len1 ? str1.charCodeAt(i) : 0;
      const c2 = i < len2 ? str2.charCodeAt(i) : 0;
      // Compare character codes using XOR
      // JavaScript strings are UTF-16, so we compare character codes
      diff |= c1 ^ c2;
    }

    // Return true only if no differences found AND lengths match
    return diff === 0 && lengthMismatch === 0;
  }

  /**
   * Constant-time buffer comparison
   *
   * Compares two buffers (Uint8Array or ArrayBuffer) in constant time
   * to prevent timing attacks. Always compares all bytes regardless of
   * early differences.
   *
   * @param buf1 - First buffer to compare (Uint8Array or ArrayBuffer)
   * @param buf2 - Second buffer to compare (Uint8Array or ArrayBuffer)
   * @returns True if buffers are equal, false otherwise
   * @throws Error if inputs are null, undefined, or not valid buffer types
   */
  static constantTimeCompareBuffers(
    buf1: Uint8Array | ArrayBuffer,
    buf2: Uint8Array | ArrayBuffer,
  ): boolean {
    // Validate inputs
    if (buf1 === null || buf1 === undefined) {
      throw new Error("First argument must be a Uint8Array or ArrayBuffer");
    }
    if (buf2 === null || buf2 === undefined) {
      throw new Error("Second argument must be a Uint8Array or ArrayBuffer");
    }

    // Convert ArrayBuffer to Uint8Array if needed
    let arr1: Uint8Array;
    let arr2: Uint8Array;

    if (buf1 instanceof ArrayBuffer) {
      arr1 = new Uint8Array(buf1);
    } else if (buf1 instanceof Uint8Array) {
      arr1 = buf1;
    } else {
      throw new Error("First argument must be a Uint8Array or ArrayBuffer");
    }

    if (buf2 instanceof ArrayBuffer) {
      arr2 = new Uint8Array(buf2);
    } else if (buf2 instanceof Uint8Array) {
      arr2 = buf2;
    } else {
      throw new Error("Second argument must be a Uint8Array or ArrayBuffer");
    }

    const len1 = arr1.length;
    const len2 = arr2.length;

    // Track length mismatch (but continue comparison)
    let lengthMismatch = len1 !== len2 ? 1 : 0;

    // Always process the maximum length to ensure constant-time behavior
    // This prevents timing leaks based on buffer length differences
    let diff = 0;
    const maxLen = Math.max(len1, len2);

    for (let i = 0; i < maxLen; i++) {
      // Get bytes, using 0 for out-of-bounds access
      // This ensures we always process the same number of iterations
      const b1 = i < len1 ? arr1[i] : 0;
      const b2 = i < len2 ? arr2[i] : 0;
      // Compare bytes using XOR
      diff |= b1 ^ b2;
    }

    // Return true only if no differences found AND lengths match
    return diff === 0 && lengthMismatch === 0;
  }
}
