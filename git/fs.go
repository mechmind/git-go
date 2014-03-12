package git

import (
	"io"
)

type FsFileAbstraction interface {
	Name() string
	io.ReadWriteCloser
}

type FsAbstraction interface {
	Open(path string) (FsFileAbstraction, error)
	Create(path string) (FsFileAbstraction, error)
	TempFile() (FsFileAbstraction, error)
	Move(from string, to string) error
	ListDir(path string) ([]string, error)
	IsFileExist(path string) bool
}
