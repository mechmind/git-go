package fsstor

import (
	"io"
)

type File interface {
	Name() string
	io.ReadWriteCloser
}

type FS interface {
	Open(path string) (File, error)
	Create(path string) (File, error)
	TempFile() (File, error)
	Move(from string, to string) error
	ListDir(path string) ([]string, error)
	IsFileExist(path string) bool

	IsReadOnly() bool
}
