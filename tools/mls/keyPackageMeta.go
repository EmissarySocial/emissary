package mls

import "encoding/hex"

// ── Parsed metadata structures ───────────────────────────────────────────────

type KeyPackageMeta struct {
	CipherSuite     string
	InitKey         keyBytes
	LeafNode        leafNodeMeta
	Signature       keyBytes
	SignatureStatus SignatureResult
}

type leafNodeMeta struct {
	Lifetime        lifetimeMeta
	EncryptionKey   keyBytes
	SignatureKey    keyBytes
	Credential      credentialMeta
	Capabilities    capabilitiesMeta
	Signature       keyBytes
	SignatureStatus SignatureResult
}

type lifetimeMeta struct {
	NotBefore uint64
	NotAfter  uint64
}

type credentialMeta struct {
	Type     string
	Identity string
}

type capabilitiesMeta struct {
	Versions     []string
	CipherSuites []string
	Extensions   []string
	Proposals    []string
	Credentials  []string
}

type keyBytes struct {
	Hex   string
	Bytes int
}

func asKeyBytes(b []byte) keyBytes {
	return keyBytes{Hex: hex.EncodeToString(b), Bytes: len(b)}
}

// parsedLeaf carries the raw byte material needed for signature verification
// alongside the display metadata.
type parsedLeaf struct {
	meta   leafNodeMeta
	encKey []byte
	sigKey []byte
	tbs    []byte
	sig    []byte
}
