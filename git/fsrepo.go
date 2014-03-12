package git

import (
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

const HEADER_BUFFER = 30

var ErrObjectNotFound = errors.New("object not found")

type FSRepo struct {
	fs FsAbstraction
}

func OpenRepo(gitDir string) (Repo, error) {
	return OpenFsRepo(&OsFs{gitDir})
}

func OpenFsRepo(fs FsAbstraction) (Repo, error) {
	return &FSRepo{fs}, nil
}

func (r *FSRepo) OpenObject(hash string) (ObjectInfo, io.ReadCloser, error) {
	// only opens loose objects now
	path := filepath.Join("objects", hash[:2], hash[2:])
	file, err := r.fs.OpenFile(path)
	if err != nil {
		return ObjectInfo{}, nil, err
	}

	reader, err := zlib.NewReader(file)
	if err != nil {
		return ObjectInfo{}, nil, err
	}

	objectInfo, err := readHeader(reader)
	if err != nil {
		reader.Close()
		return ObjectInfo{}, nil, err
	}

	objectInfo.Hash = hash

	return objectInfo, newObjectReader(reader, objectInfo.Size), nil
}

func (r *FSRepo) CreateObject(objType int8, size uint64) (ObjectWriter, error) {
	var objTypeName string
	switch objType {
	case TYPE_BLOB:
		objTypeName = "blob"
	case TYPE_COMMIT:
		objTypeName = "commit"
	case TYPE_TAG:
		objTypeName = "tag"
	case TYPE_TREE:
		objTypeName = "tree"
	default:
		return nil, errors.New("unknown object type")
	}

	// create header
	header := make([]byte, HEADER_BUFFER)
	pos := copy(header, []byte(objTypeName))
	header[pos] = ' '
	pos++
	pos += copy(header[pos:], []byte(strconv.FormatUint(size, 10)))
	header[pos] = 0
	header = header[:pos+1]

	tmp, err := r.fs.TmpFile()
	if err != nil {
		return nil, err
	}

	writer := newObjectWriter(tmp, size+uint64(len(header)), r)

	if _, err = writer.Write(header); err != nil {
		writer.closeWriters()
		return nil, err
	}

	return writer, nil
}

func (r *FSRepo) ReadRef(ref string) (string, error) {
	// read refs till object found
	for {
		value, err := readRefFile(r.fs, ref)
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(value, "ref: ") {
			// symbolic ref, following
			ref = value[5:]
		} else {
			// found hash
			return value, nil
		}
	}
}

func (r *FSRepo) UpdateRef(ref, value string) error {
	// update existing ref or create one
	// FIXME: check that value is a hash
	return writeRefFile(r.fs, ref, value)
}

func (r *FSRepo) ReadSymbolicRef(ref string) (string, error) {
	// read symbolic ref. Returns error if ref is not symbolic
	value, err := readRefFile(r.fs, ref)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(value, "ref: ") {
		return value[5:], nil
	} else {
		return "", ERR_NOT_A_SYMBOLIC_REF
	}
}

func (r *FSRepo) UpdateSymbolicRef(ref, value string) error {
	// update symbolic reference or create one
	// FIXME: check that value is an existing ref?
	return writeRefFile(r.fs, ref, "ref: "+value)
}

func (r *FSRepo) ListRefs(kind int) ([]string, error) {
	switch kind {
	case KIND_BRANCH:
		return r.fs.ListDir("refs/heads")
	case KIND_TAG:
		return r.fs.ListDir("refs/tags")
	default:
		return nil, errors.New("invalid ref kind")
	}
}

func (r *FSRepo) IsObjectExists(hash string) bool {
	return r.fs.IsFileExist(filepath.Join("objects", hash))
}

func (r *FSRepo) insertObject(hash string, src FsFileAbstraction) error {
	path := filepath.Join("objects", hash[:2], hash[2:])
	dst, err := r.fs.EnsureFile(path)
	if err != nil {
		return err
	}

	return r.fs.Move(src, dst)
}

type objectReader struct {
	source io.ReadCloser
	io.Reader
}

func (o objectReader) Close() error {
	return o.source.Close()
}

func newObjectReader(source io.ReadCloser, size uint64) objectReader {
	// FIXME: reimplement limitreader to use uint64 for strict compatibility
	return objectReader{source, io.LimitReader(source, int64(size))}
}

type objectWriter struct {
	repo       *FSRepo
	file       FsFileAbstraction
	zlibWriter *zlib.Writer
	hashWriter hash.Hash
	id         string
	io.WriteCloser
}

func newObjectWriter(file FsFileAbstraction, size uint64, repo *FSRepo) *objectWriter {
	zw := zlib.NewWriter(file)
	hw := sha1.New()
	allw := &exactSizeWriter{size, io.MultiWriter(hw, zw)}
	return &objectWriter{repo, file, zw, hw, "", allw}
}

func (ob *objectWriter) closeWriters() error {
	//err := ob.file.Close()
	//if err != nil {
	//    return err
	//}

	err := ob.WriteCloser.Close()
	if err != nil {
		return err
	}

	err = ob.zlibWriter.Close()
	if err != nil {
		return err
	}

	return nil
}

func (ob *objectWriter) Close() error {
	defer ob.file.Close()
	err := ob.closeWriters()
	if err != nil {
		return err
	}

	ob.id = fmt.Sprintf("%x", ob.hashWriter.Sum(nil))

	// insert object into repo storage
	return ob.repo.insertObject(ob.id, ob.file)
}

func (ob *objectWriter) Id() string {
	return ob.id
}

func readHeader(src io.Reader) (ObjectInfo, error) {
	var objectInfo ObjectInfo
	var buf = make([]byte, HEADER_BUFFER)

	// read type of object
	obuf, err := scanUntil(src, ' ', buf)
	if err != nil {
		return objectInfo, err
	}
	objType := string(obuf)
	switch objType {
	case "blob":
		objectInfo.Type = TYPE_BLOB
	case "commit":
		objectInfo.Type = TYPE_COMMIT
	case "tag":
		objectInfo.Type = TYPE_TAG
	case "tree":
		objectInfo.Type = TYPE_TREE
	default:
		return objectInfo, errors.New("unknown object type")
	}

	// read length of object
	obuf, err = scanUntil(src, 0, buf)
	if err != nil {
		return ObjectInfo{}, err
	}

	size, err := strconv.ParseUint(string(obuf), 10, 64)
	if err != nil {
		return ObjectInfo{}, err
	}
	objectInfo.Size = size
	return objectInfo, nil
}

// scan src until needle is encountered or buf is depleted
func scanUntil(src io.Reader, needle byte, buf []byte) ([]byte, error) {
	buf = buf[:0]
	for i := 0; i < cap(buf)-1; i++ {
		_, err := src.Read(buf[i : i+1])
		if err != nil {
			return nil, err
		}
		if buf[i : i+1][0] == needle {
			return buf[:i], nil
		}
	}
	return nil, errors.New("buffer depleted")
}

func readRefFile(fs FsAbstraction, path string) (string, error) {
	file, err := fs.OpenFile(path)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(string(data))
	return value, nil
}

func writeRefFile(fs FsAbstraction, path string, data string) error {
	file, err := fs.EnsureFile(path)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write([]byte(data))

	return err
}
