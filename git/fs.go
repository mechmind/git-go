package git

import (
	"io"
)

type FsFileAbstraction interface {
	Name() string
	io.ReadWriteCloser
}

type FsAbstraction interface {
	OpenFile(path string) (FsFileAbstraction, error)
	EnsureFile(path string) (FsFileAbstraction, error)
	TmpFile() (FsFileAbstraction, error)
	Move(from FsFileAbstraction, to FsFileAbstraction) error
	ListDir(path string) ([]string, error)
	IsFileExist(path string) bool
}
