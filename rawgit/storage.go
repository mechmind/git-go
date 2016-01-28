package rawgit

import (
	"io"
)

type Storage interface {
	OpenObject(oid *OID) (ObjectInfo, io.ReadCloser, error)
	StatObject(oid *OID) (ObjectInfo, interface{}, error)
	CreateObject(objType OType, size uint64) (ObjectWriter, error)
	IsObjectExist(oid *OID) bool

	ListRefs(ns string) ([]string, error)
	ReadRef(name string) (string, error)
	ResolveRef(name string) (*OID, error)
	ReadSpecialRef(ref SpecialRef) (string, error)
	UpdateRef(name, value string) error
	UpdateSpecialRef(ref SpecialRef, value string) error

	IsReadOnly() bool
}
