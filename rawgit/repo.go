package rawgit

import (
	"encoding/hex"
	"io"
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

type OID [20]byte

func (oid *OID) String() string {
	return hex.EncodeToString(oid[:])
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

type ObjectInfo struct {
	OID
	OType
	Size uint64
}

// Repo functions:
// * raw object access (read, write, lookup)
// * ref resolving
// * history lookup?
type Repo interface {
	// raw object operations
	OpenObject(oid *OID) (ObjectInfo, io.ReadCloser, error)
	StatObject(oid *OID) (ObjectInfo, interface{}, error)
	CreateObject(objType OType, size uint64) (ObjectWriter, error)
	IsObjectExists(oid *OID) bool

	// ref operations
	ReadRef(name string) (string, error)
	UpdateRef(src, dest string) error

	// symbolic ref operations
	ReadSymbolicRef(name string) (string, error)
	UpdateSymbolicRef(src, dest string) error

	// ref list
	ListRefs(ns string) ([]string, error)
}

type ObjectWriter interface {
	io.WriteCloser
	OID() *OID
}
