package mls

import "errors"

const (
	protocolVersionMLS10     = uint16(1)
	wireFormatMLSKeyPackage  = uint16(5)
	leafNodeSourceKeyPackage = uint8(1)
	credentialTypeBasic      = uint16(1)
	credentialTypeX509       = uint16(2)
)

// ── Known identifier names ───────────────────────────────────────────────────

var cipherSuiteNames = map[uint16]string{
	0x0001: "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519",
	0x0002: "MLS_128_DHKEMP256_AES128GCM_SHA256_P256",
	0x0003: "MLS_128_DHKEMX25519_CHACHA20POLY1305_SHA256_Ed25519",
	0x0004: "MLS_256_DHKEMX448_AES256GCM_SHA512_Ed448",
	0x0005: "MLS_256_DHKEMP521_AES256GCM_SHA512_P521",
	0x0006: "MLS_256_DHKEMX448_CHACHA20POLY1305_SHA512_Ed448",
	0x0007: "MLS_256_DHKEMP384_AES256GCM_SHA384_P384",
}

var extensionTypeNames = map[uint16]string{
	0x0001: "application_id",
	0x0002: "ratchet_tree",
	0x0003: "required_capabilities",
	0x0004: "external_pub",
	0x0005: "external_senders",
}

var proposalTypeNames = map[uint16]string{
	0x0001: "add",
	0x0002: "update",
	0x0003: "remove",
	0x0004: "psk",
	0x0005: "reinit",
	0x0006: "external_init",
	0x0007: "group_context_extensions",
}

var credentialTypeNames = map[uint16]string{
	0x0001: "basic",
	0x0002: "x509",
}

var wireFormatNames = map[uint16]string{ //nolint:unused // retained for future use
	0x0000: "RESERVED",
	0x0001: "mls_public_message",
	0x0002: "mls_private_message",
	0x0003: "mls_welcome",
	0x0004: "mls_group_info",
	0x0005: "mls_key_package",
}

var errNonMinimalVarint = errors.New("mls: non-minimal varint encoding (forbidden by RFC 9420 §2.1.2)")
