package git

import (
	"errors"
	"io"
)

// mirrored from git/cache.h
const TYPE_BAD = -1
const (
	TYPE_NONE = iota
	TYPE_COMMIT
	TYPE_TREE
	TYPE_BLOB
	TYPE_TAG
	TYPE_unset
	TYPE_OFS_DELTA
	TYPE_REF_DELTA
	TYPE_ANY
	TYPE_MAX
)

var (
	ERR_INVALID_REF        = errors.New("invalid reference")
	ERR_NOT_A_SYMBOLIC_REF = errors.New("not a symbolic reference")
)

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
