package git

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type FileStore struct {
	gitDir string
}

type tmpObjectWriter struct {
	file *os.File
}

func (t tmpObjectWriter) Read(p []byte) (int, error) {
	return t.file.Read(p)
}

func (t tmpObjectWriter) Write(p []byte) (int, error) {
	return t.file.Write(p)
}

func (t tmpObjectWriter) Close() error {
	err := t.file.Close()
	os.Remove(t.file.Name())
	return err
}

func NewFileStore(dir string) Store {
	return FileStore{dir}
}

func (o FileStore) NewRawObjectWriter() (io.ReadWriteCloser, error) {
	tmpFile, err := ioutil.TempFile(o.gitDir, "tmpgitgo.")
	if err != nil {
		return nil, err
	}

	return tmpObjectWriter{file: tmpFile}, nil
}

func (o FileStore) NewRawObjectReader(hash string) (io.ReadCloser, error) {
	path := filepath.Join(o.gitDir, "objects", hash[:2], hash[2:])

	return os.Open(path)
}

func (o FileStore) ReadRef(ref string) (string, error) {
	path := filepath.Join(o.gitDir, ref)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(file)
	return strings.TrimSpace(string(data)), err
}

func (o FileStore) WriteRef(ref string, contents string) error {
	path := filepath.Join(o.gitDir, ref)
	dir := filepath.Dir(path)
	_, err := os.Stat(dir)
	switch {
	case os.IsNotExist(err):
		// make ref dirs
		err2 := os.MkdirAll(dir, os.ModeDir|0777)
		if err2 != nil {
			return err2
		}
	case err != nil:
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	_, err = file.WriteString(contents)
	if err != nil {
		return err
	}

	return file.Close()
}

func (o FileStore) ListRefs(ns string) ([]string, error) {
	baseDir := filepath.Join(o.gitDir, "refs")
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
		names[idx] = info.Name()
	}
	return names, nil
}

func (o FileStore) StoreObject(hash string, r io.ReadCloser) error {
	dir := filepath.Join(o.gitDir, "objects", hash[:2])
	objectFileName := filepath.Join(dir, hash[2:])
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}

	switch t := r.(type) {
	case tmpObjectWriter:
		fmt.Println(objectFileName)
		return os.Rename(t.file.Name(), objectFileName)
	default:
		panic("not supported")
	}

	return nil
}
