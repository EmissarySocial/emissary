package mls

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/benpate/derp"
)

// isGrease reports whether v is one of the GREASE values reserved by
// RFC 9420 §13.5 (0x0A0A, 0x1A1A, ..., 0xEAEA). Clients deliberately
// advertise these in Capabilities; they are expected, not errors.
func isGrease(v uint16) bool {
	return v >= 0x0A0A && v <= 0xEAEA && (v-0x0A0A)%0x1010 == 0
}

// lookupOrHex looks up id in m, returning a string containing the name and hex value if found, or just the hex value if not found
// GREASE values are labeled as such.
func lookupOrHex(m map[uint16]string, id uint16) string {
	if n, ok := m[id]; ok {
		return fmt.Sprintf("%s (0x%04x)", n, id)
	}
	if isGrease(id) {
		return fmt.Sprintf("GREASE (0x%04x)", id)
	}
	return fmt.Sprintf("0x%04x", id)
}

// formatCipherSuite returns a human-friendly name for the given cipher suite ID, or a hex string if unknown.
func formatCipherSuite(id uint16) string {
	if n, ok := cipherSuiteNames[id]; ok {
		return fmt.Sprintf("%s (0x%04x)", n, id)
	}
	return fmt.Sprintf("unknown (0x%04x)", id)
}

// writeVarLenVec writes a byte slice prefixed by its length as an MLS variable-length integer (RFC 9420 §2.1.2).
func writeVarLenVec(b *bytes.Buffer, v []byte) error {
	n := uint64(len(v))
	switch {
	case n < 0x40:
		b.WriteByte(byte(n))
	case n < 0x4000:
		var t [2]byte
		binary.BigEndian.PutUint16(t[:], uint16(n)|0x4000)
		b.Write(t[:])
	case n < 0x40000000:
		var t [4]byte
		binary.BigEndian.PutUint32(t[:], uint32(n)|0x80000000)
		b.Write(t[:])
	default:
		return derp.Internal("mls.writeVarLenVec", "vector too long", n)
	}
	b.Write(v)
	return nil
}

// signContent builds the SignContent structure from RFC 9420 §5.1.2:
func signContent(label string, content []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := writeVarLenVec(&b, []byte("MLS 1.0 "+label)); err != nil {
		return nil, err
	}
	if err := writeVarLenVec(&b, content); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// mapUint16s looks up each uint16 in vs in names, returning a slice of the corresponding names or hex strings if not found.
func mapUint16s(vs []uint16, names map[uint16]string) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = lookupOrHex(names, v)
	}
	return out
}

// printableOrHex returns a string containing the UTF-8 text of b if all bytes are in the printable ASCII range, or a hex encoding otherwise.
func printableOrHex(b []byte) string {
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return hex.EncodeToString(b)
		}
	}
	return string(b)
}
