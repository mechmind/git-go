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
    CreateObject(size int64) (io.WriteCloser, error)
    IsObjectExists(hash string) bool
}


