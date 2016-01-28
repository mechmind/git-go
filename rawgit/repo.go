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

type OID [20]byte

func (oid *OID) String() string {
	return hex.EncodeToString(oid[:])
}

type Object interface {
	OID() OID
	OType() OType
}

type ObjectInfo struct {
	Type int8
	Hash string
	Size uint64
}

// Repo functions:
// * raw object access (read, write, lookup)
// * ref resolving
// * history lookup?
type Repo interface {
	// raw object operations
	OpenObject(hash string) (ObjectInfo, io.ReadCloser, error)
	CreateObject(objType int8, size uint64) (ObjectWriter, error)
	IsObjectExists(hash string) bool

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
	Id() string
}
