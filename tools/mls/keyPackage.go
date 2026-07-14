package mls

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/benpate/derp"
)

// ParseMLSKeyPackage parses an MLS KeyPackage (RFC 9420 §10)
// and verifies its signatures where the cipher suite's signature scheme is
// supported (Ed25519 suites 0x0001 and 0x0003).
//
// Wire format:
//
//	KeyPackage { version uint16, cipher_suite uint16, init_key<V>,
//	             leaf_node, extensions<V>, signature<V> }
func ParseKeyPackage(content string) (*KeyPackageMeta, error) {

	const location = "mls.ParseKeyPackage"

	// Parse base64-encoded input
	raw, err := base64.StdEncoding.DecodeString(content)

	if err != nil {
		return nil, derp.Wrap(err, location, "invalid base64")
	}

	r := newMLSReader(raw)

	// KeyPackageTBS covers everything from KeyPackage.version up to (but not
	// including) the signature (RFC 9420 §10).

	kpVersion, err := r.readUint16()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading KeyPackage version")
	}
	if kpVersion != protocolVersionMLS10 {
		return nil, derp.Internal(location, "unsupported KeyPackage version (expected 0x0001)", kpVersion)
	}
	cipherSuite, err := r.readUint16()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading cipher_suite")
	}
	initKey, err := r.readOpaqueVec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading init_key")
	}
	leaf, err := parseLeafNode(r)
	if err != nil {
		return nil, derp.Wrap(err, location, "reading leaf_node")
	}
	// RFC 9420 §10.1: init_key MUST differ from leaf_node.encryption_key.
	if bytes.Equal(initKey, leaf.encKey) {
		return nil, derp.Internal(location, "invalid KeyPackage: init_key equals leaf_node.encryption_key (RFC 9420 §10.1)")
	}
	if _, err = r.readOpaqueVec(); err != nil { // Extension extensions<V> — skip content
		return nil, derp.Wrap(err, location, "reading KeyPackage extensions")
	}
	kpTBS := r.data[:r.pos]
	sig, err := r.readOpaqueVec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading KeyPackage signature")
	}
	if r.remaining() != 0 {
		return nil, derp.Internal(location, "malformed KeyPackage: trailing bytes after signature", r.remaining())
	}

	meta := &KeyPackageMeta{
		CipherSuite:     formatCipherSuite(cipherSuite),
		InitKey:         asKeyBytes(initKey),
		LeafNode:        leaf.meta,
		Signature:       asKeyBytes(sig),
		SignatureStatus: verifySignature(cipherSuite, leaf.sigKey, "KeyPackageTBS", kpTBS, sig),
	}
	meta.LeafNode.SignatureStatus = verifySignature(cipherSuite, leaf.sigKey, "LeafNodeTBS", leaf.tbs, leaf.sig)
	return meta, nil
}

// parseLeafNode parses a LeafNode (RFC 9420 §7.2). Within a KeyPackage, the
// leaf_node_source MUST be key_package (RFC 9420 §10.1), so this parser
// rejects update/commit sources rather than handling their variant fields.
// For source key_package, LeafNodeTBS is exactly the LeafNode content
// preceding the signature (no group_id/leaf_index are appended).
func parseLeafNode(r *mlsReader) (*parsedLeaf, error) {

	const location = "mls.parseLeafNode"

	start := r.pos

	encKey, err := r.readOpaqueVec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading encryption_key")
	}
	sigKey, err := r.readOpaqueVec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading signature_key")
	}
	cred, err := parseCredential(r)
	if err != nil {
		return nil, derp.Wrap(err, location, "reading credential")
	}
	caps, err := parseCapabilities(r)
	if err != nil {
		return nil, derp.Wrap(err, location, "reading capabilities")
	}
	source, err := r.readUint8()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading leaf_node_source")
	}
	if source != leafNodeSourceKeyPackage {
		return nil, derp.Internal(location, "leaf_node_source must be key_package (1) inside a KeyPackage (RFC 9420 §10.1)", source)
	}

	// Lifetime { not_before uint64, not_after uint64 }
	notBefore, err := r.readUint64()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading lifetime.not_before")
	}
	notAfter, err := r.readUint64()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading lifetime.not_after")
	}

	if _, err = r.readOpaqueVec(); err != nil { // Extension extensions<V> — skip
		return nil, derp.Wrap(err, location, "reading leaf node extensions")
	}
	tbs := r.data[start:r.pos]
	sig, err := r.readOpaqueVec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading leaf node signature")
	}

	return &parsedLeaf{
		meta: leafNodeMeta{
			Lifetime:      lifetimeMeta{NotBefore: notBefore, NotAfter: notAfter},
			EncryptionKey: asKeyBytes(encKey),
			SignatureKey:  asKeyBytes(sigKey),
			Credential:    *cred,
			Capabilities:  *caps,
			Signature:     asKeyBytes(sig),
		},
		encKey: encKey,
		sigKey: sigKey,
		tbs:    tbs,
		sig:    sig,
	}, nil
}

// parseCredential parses a Credential (RFC 9420 §5.3).
func parseCredential(r *mlsReader) (*credentialMeta, error) {

	const location = "mls.parseCredential"

	credType, err := r.readUint16()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading credential type")
	}
	switch credType {
	case credentialTypeBasic:
		identity, err := r.readOpaqueVec()
		if err != nil {
			return nil, derp.Wrap(err, location, "reading basic identity")
		}
		return &credentialMeta{Type: "basic", Identity: printableOrHex(identity)}, nil
	case credentialTypeX509:
		// Certificate certificates<V> — a vector of Certificate structs, each
		// an opaque cert_data<V>. Reading the outer byte-length prefix and
		// skipping correctly advances past the whole chain.
		certBlob, err := r.readOpaqueVec()
		if err != nil {
			return nil, derp.Wrap(err, location, "reading x509 certificates")
		}
		return &credentialMeta{
			Type:     "x509",
			Identity: fmt.Sprintf("[certificate chain, %d bytes]", len(certBlob)),
		}, nil
	default:
		return nil, derp.Internal(location, "unsupported credential type", credType)
	}
}

// parseCapabilities parses a Capabilities struct (RFC 9420 §7.2).
func parseCapabilities(r *mlsReader) (*capabilitiesMeta, error) {

	const location = "mls.parseCapabilities"

	versions, err := r.readUint16Vec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading versions")
	}
	cipherSuites, err := r.readUint16Vec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading cipher_suites")
	}
	exts, err := r.readUint16Vec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading extensions")
	}
	proposals, err := r.readUint16Vec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading proposals")
	}
	creds, err := r.readUint16Vec()
	if err != nil {
		return nil, derp.Wrap(err, location, "reading credential_types")
	}

	fmtVersions := make([]string, len(versions))
	for i, v := range versions {
		if v == 1 {
			fmtVersions[i] = "MLS 1.0 (0x0001)"
		} else if isGrease(v) {
			fmtVersions[i] = fmt.Sprintf("GREASE (0x%04x)", v)
		} else {
			fmtVersions[i] = fmt.Sprintf("0x%04x", v)
		}
	}

	return &capabilitiesMeta{
		Versions:     fmtVersions,
		CipherSuites: mapUint16s(cipherSuites, cipherSuiteNames),
		Extensions:   mapUint16s(exts, extensionTypeNames),
		Proposals:    mapUint16s(proposals, proposalTypeNames),
		Credentials:  mapUint16s(creds, credentialTypeNames),
	}, nil
}
