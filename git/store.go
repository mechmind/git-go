package git

import (
	"io"
)

type Store interface {
	NewRawObjectWriter() (io.ReadWriteCloser, error)
	NewRawObjectReader(hash string) (io.ReadCloser, error)
	ReadRef(ref string) (string, error)
	WriteRef(ref string, contents string) error
	ListRefs(ns string) ([]string, error)
	StoreObject(hash string, r io.ReadCloser) error
}
