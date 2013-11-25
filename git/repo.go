package git

import (
    "io"
)

const (
    TYPE_BLOB = iota
    TYPE_TREE
    TYPE_COMMIT
    TYPE_TAG
    TYPE_UNKNOWN
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
}

type ObjectWriter interface {
    io.WriteCloser
    Id() string
}

