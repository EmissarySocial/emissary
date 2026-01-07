/**
 * Key Authentication Utility
 *
 * Provides key fingerprinting for MITM attack prevention.
 * Generates SHA-256 fingerprints of public keys for verification.
 */

import { ConstantTime } from "./constantTime";

/**
 * Key authentication utility for fingerprinting public keys
 */
export class KeyAuthentication {
  /**
   * Generate a fingerprint for a public key
   *
   * @param publicKey - The public key (CryptoKey or Uint8Array)
   * @returns Hex string fingerprint with colons (e.g., "aa:bb:cc:dd:...")
   * @throws Error if publicKey is null, undefined, empty, or invalid format
   */
  static async generateFingerprint(
    publicKey: CryptoKey | Uint8Array,
  ): Promise<string> {
    // Validate input is not null or undefined
    if (publicKey === null || publicKey === undefined) {
      throw new Error("Public key is required");
    }

    let keyBytes: Uint8Array;

    // Check if it's a CryptoKey by checking for Web Crypto API key properties
    if (
      publicKey &&
      typeof publicKey === "object" &&
      "type" in publicKey &&
      "algorithm" in publicKey
    ) {
      try {
        const exported = await crypto.subtle.exportKey(
          "raw",
          publicKey as CryptoKey,
        );
        keyBytes = new Uint8Array(exported);
      } catch (error) {
        // Don't leak sensitive data in error messages
        const errorMessage =
          error instanceof Error ? error.message : String(error);
        if (
          errorMessage.includes("exportKey") ||
          errorMessage.includes("key")
        ) {
          throw new Error("Invalid key format: key cannot be exported");
        }
        throw new Error(`Invalid key format: ${errorMessage}`);
      }
    } else {
      // Validate Uint8Array format
      if (!(publicKey instanceof Uint8Array)) {
        throw new Error("Public key must be a CryptoKey or Uint8Array");
      }

      keyBytes = publicKey as Uint8Array;

      // Validate Uint8Array is not empty
      if (keyBytes.length === 0) {
        throw new Error("Public key cannot be empty");
      }

      // Validate reasonable maximum size (10KB for public keys)
      // This prevents DoS attacks with extremely large keys
      const MAX_KEY_SIZE = 10240; // 10KB
      if (keyBytes.length > MAX_KEY_SIZE) {
        throw new Error(
          `Public key too large: maximum size is ${MAX_KEY_SIZE} bytes`,
        );
      }
    }

    // Hash the key to create fingerprint
    const hash = await crypto.subtle.digest("SHA-256", keyBytes);
    const hashArray = Array.from(new Uint8Array(hash));

    // Format as hex string with colons
    return hashArray.map((b) => b.toString(16).padStart(2, "0")).join(":");
  }

  /**
   * Verify key fingerprint matches expected value
   *
   * @param publicKey - The public key to verify
   * @param expectedFingerprint - The expected fingerprint
   * @returns True if fingerprint matches
   * @throws Error if inputs are invalid
   */
  static async verifyFingerprint(
    publicKey: CryptoKey | Uint8Array,
    expectedFingerprint: string,
  ): Promise<boolean> {
    // Validate inputs
    if (publicKey === null || publicKey === undefined) {
      throw new Error("Public key is required");
    }

    if (expectedFingerprint === null || expectedFingerprint === undefined) {
      throw new Error("Expected fingerprint is required");
    }

    if (typeof expectedFingerprint !== "string") {
      throw new Error("Expected fingerprint must be a string");
    }

    if (expectedFingerprint.length === 0) {
      throw new Error("Expected fingerprint cannot be empty");
    }

    const fingerprint = await this.generateFingerprint(publicKey);
    // Use constant-time comparison to prevent timing attacks
    return ConstantTime.constantTimeCompareStrings(
      fingerprint,
      expectedFingerprint,
    );
  }
}
