package fsstor

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type OSFS struct {
	root string
}

func (o OSFS) Open(path string) (File, error) {
	return os.Open(filepath.Join(o.root, path))
}

func (o OSFS) Create(path string) (File, error) {
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

func (o OSFS) TempFile() (File, error) {
	tmp, err := ioutil.TempFile(o.root, "tmpgitgo.")
	if err != nil {
		return nil, err
	}

	return &tmpFileRemover{tmp}, nil
}

func (o OSFS) Move(from string, to string) error {
	dst, err := o.Create(filepath.Join(o.root, to))
	if err != nil {
		return err
	}

	return os.Rename(from, dst.Name())
}

func (o OSFS) ListDir(path string) ([]string, error) {
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

func (o OSFS) IsFileExist(path string) bool {
	if _, err := os.Stat(filepath.Join(o.root, path)); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func (o OSFS) IsReadOnly() bool {
	// TODO: check for write permissions
	return false
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
