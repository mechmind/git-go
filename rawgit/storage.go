package rawgit

import (
	"io"
)

type FsFile interface {
	Name() string
	io.ReadWriteCloser
}

type Fs interface {
	Open(path string) (FsFile, error)
	Create(path string) (FsFile, error)
	TempFile() (FsFile, error)
	Move(from string, to string) error
	ListDir(path string) ([]string, error)
	IsFileExist(path string) bool
}
