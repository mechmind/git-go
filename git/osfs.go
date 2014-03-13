package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type OsFs struct {
	root string
}

func (o OsFs) Open(path string) (FsFileAbstraction, error) {
	return os.Open(filepath.Join(o.root, path))
}

func (o OsFs) Create(path string) (FsFileAbstraction, error) {
	path = filepath.Join(o.root, path)
	base := filepath.Dir(path)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		err := os.MkdirAll(base, 0755)
		if err != nil {
			return nil, err
		}
	}

	return os.Create(path)
}

func (o OsFs) TempFile() (FsFileAbstraction, error) {
	tmp, err := ioutil.TempFile(o.root, "tmpgitgo.")
	if err != nil {
		return nil, err
	}

	return &tmpFileRemover{tmp}, nil
}

func (o OsFs) Move(from string, to string) error {
	dst, err := o.Create(filepath.Join(o.root, to))
	if err != nil {
		return err
	}

	return os.Rename(from, dst.Name())
}

func (o OsFs) ListDir(path string) ([]string, error) {
	baseDir := filepath.Join(o.root, path)
	// FIXME: refs with slashes
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		// no dir, no refs
		return nil, nil
	}

	fileList, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(fileList))
	for idx, info := range fileList {
		names[idx] = filepath.Join(path, info.Name())
	}
	return names, nil
}

func (o OsFs) IsFileExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

type tmpFileRemover struct {
	*os.File
}

func (t tmpFileRemover) Close() error {
	err := t.File.Close()
	if err != nil {
		return err
	}

	return os.Remove(t.File.Name())
}
