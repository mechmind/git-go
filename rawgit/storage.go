package rawgit

import (
	"io"
)

type Storage interface {
	OpenObject(oid *OID) (ObjectInfo, io.ReadCloser, error)
	StatObject(oid *OID) (ObjectInfo, interface{}, error)
	CreateObject(objType OType, size uint64) (ObjectWriter, error)
	IsObjectExist(oid *OID) bool
}

type RefDatabase interface {
	ListRefs(ns string) ([]string, error)
	ReadRef(name string) (string, error)
	ReadSpecialRef(ref SpecialRef) (string, error)
	WriteRef(name, value string) error
	WriteSpecialRef(ref SpecialRef, value string) error
}

type ReadOnly interface {
	IsReadOnly() bool
}
