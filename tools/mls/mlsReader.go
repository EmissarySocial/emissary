package mls

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/benpate/derp"
)

// ── MLS reader ───────────────────────────────────────────────────────────────
// All vector lengths in MLS use the QUIC variable-length integer encoding
// (RFC 9000 §16), restricted per RFC 9420 §2.1.2: the 8-byte encoding is
// forbidden (max length 2^30-1) and non-minimal encodings MUST be rejected.

type mlsReader struct {
	data []byte
	pos  int
}

func newMLSReader(data []byte) *mlsReader { return &mlsReader{data: data} }
func (r *mlsReader) remaining() int       { return len(r.data) - r.pos }

func (r *mlsReader) readUint8() (uint8, error) {
	if r.remaining() < 1 {
		return 0, io.ErrUnexpectedEOF
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

func (r *mlsReader) readUint16() (uint16, error) {
	if r.remaining() < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v, nil
}

func (r *mlsReader) readUint64() (uint64, error) {
	if r.remaining() < 8 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v, nil
}

// readVarLen reads an MLS variable-length integer (RFC 9420 §2.1.2).
// The two high bits of the first byte encode the total byte count:
//
//	00 → 1 byte  (6-bit value,  max 63)
//	01 → 2 bytes (14-bit value, max 16383)
//	10 → 4 bytes (30-bit value, max 2^30-1)
//	11 → forbidden in MLS (QUIC's 8-byte form)
//
// Non-minimal encodings are rejected as malformed.
func (r *mlsReader) readVarLen() (uint64, error) {
	if r.remaining() < 1 {
		return 0, io.ErrUnexpectedEOF
	}
	prefix := r.data[r.pos] >> 6
	switch prefix {
	case 0:
		v := uint64(r.data[r.pos] & 0x3f)
		r.pos++
		return v, nil
	case 1:
		if r.remaining() < 2 {
			return 0, io.ErrUnexpectedEOF
		}
		v := uint64(binary.BigEndian.Uint16(r.data[r.pos:])) & 0x3fff
		r.pos += 2
		if v < 0x40 {
			return 0, errNonMinimalVarint
		}
		return v, nil
	case 2:
		if r.remaining() < 4 {
			return 0, io.ErrUnexpectedEOF
		}
		v := uint64(binary.BigEndian.Uint32(r.data[r.pos:])) & 0x3fffffff
		r.pos += 4
		if v < 0x4000 {
			return 0, errNonMinimalVarint
		}
		return v, nil
	default: // prefix 0b11
		return 0, errors.New("mls: 8-byte varint encoding is not permitted by RFC 9420")
	}
}

// readOpaqueVec reads a VarLen-prefixed byte vector.
// Safe on 32-bit platforms: readVarLen caps values at 2^30-1.
func (r *mlsReader) readOpaqueVec() ([]byte, error) {
	n, err := r.readVarLen()
	if err != nil {
		return nil, err
	}
	if r.remaining() < int(n) {
		return nil, io.ErrUnexpectedEOF
	}
	v := make([]byte, n)
	copy(v, r.data[r.pos:r.pos+int(n)])
	r.pos += int(n)
	return v, nil
}

// readUint16Vec reads a VarLen-prefixed vector of uint16 values.
func (r *mlsReader) readUint16Vec() ([]uint16, error) {
	byteLen, err := r.readVarLen()
	if err != nil {
		return nil, err
	}
	if byteLen%2 != 0 {
		return nil, derp.Internal("mls.readUint16Vec", "uint16 vector has odd byte length", byteLen)
	}
	out := make([]uint16, byteLen/2)
	for i := range out {
		v, err := r.readUint16()
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}
