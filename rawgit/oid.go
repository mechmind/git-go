package rawgit

import (
	"encoding/hex"
)

type OType int8

// mirrored from git/cache.h
const OTypeBad = -1
const (
	OTypeNone = iota
	OTypeCommit
	OTypeTree
	OTypeBlob
	OTypeTag
	OTypeUnset
	OTypeOffsetDelta
	OTypeRefDelta
	OTypeAny
)

func (ot OType) String() string {
	switch ot {
	case OTypeBad:
		return "<bad>"
	case OTypeNone:
		return "<none>"
	case OTypeCommit:
		return "commit"
	case OTypeBlob:
		return "blob"
	case OTypeTag:
		return "tag"
	case OTypeUnset:
		return "<unset>"
	case OTypeOffsetDelta:
		return "<offset delta>"
	case OTypeRefDelta:
		return "<ref delta>"
	case OTypeAny:
		return "<any>"
	default:
		return "<invalid>"
	}
}

// for embedding
func (ot OType) GetOType() OType {
	return ot
}

func ParseOType(ot string) OType {
	switch ot {
	case "blob":
		return OTypeBlob
	case "tree":
		return OTypeTree
	case "commit":
		return OTypeCommit
	case "tag":
		return OTypeTag
	default:
		return OTypeBad
	}
}

type OID [20]byte

func (oid *OID) String() string {
	return hex.EncodeToString(oid[:])
}

func (oid *OID) Equal(comp *OID) bool {
	return *oid == *comp
}

// for embedding
func (oid *OID) GetOID() *OID {
	return oid
}

func ParseOID(src string) (*OID, error) {
	if len(src) != 40 {
		return nil, ErrInvalidHashLength
	}

	buf, err := hex.DecodeString(src)
	if err != nil {
		return nil, err
	}

	oid := OID{}
	copy(oid[:], buf)

	return &oid, nil
}

func OIDFromBytes(buf []byte) (*OID, error) {
	if len(buf) != 20 {
		return nil, ErrInvalidHashLength
	}

	oid := OID{}
	copy(oid[:], buf)
	return &oid, nil
}

type ObjectInfo struct {
	OID
	OType
	Size uint64
}
