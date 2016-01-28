package rawgit

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
	fs    Fs
	packs map[string]*Pack
}

func OpenRepo(gitDir string) (Repo, error) {
	return OpenFsRepo(&OsFs{gitDir})
}

func OpenFsRepo(fs Fs) (Repo, error) {
	repo := &FSRepo{fs, make(map[string]*Pack)}
	// load packs
	err := repo.scanPacks()
	return repo, err
}

func (r *FSRepo) OpenObject(hash string) (ObjectInfo, io.ReadCloser, error) {
	path := filepath.Join("objects", hash[:2], hash[2:])
	if !r.fs.IsFileExist(path) {
		// lookup object in packs
		for _, pack := range r.packs {
			if pack.HasObject(hash) {
				return pack.OpenObject(hash)
			}
		}
		return ObjectInfo{}, nil, ErrObjectNotFound
	}
	file, err := r.fs.Open(path)
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
	case OTypeBlob:
		objTypeName = "blob"
	case OTypeCommit:
		objTypeName = "commit"
	case OTypeTag:
		objTypeName = "tag"
	case OTypeTree:
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

	tmp, err := r.fs.TempFile()
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

func (r *FSRepo) ListRefs(ns string) ([]string, error) {
	return r.fs.ListDir(filepath.Join("refs", ns))
}

func (r *FSRepo) IsObjectExists(hash string) bool {
	return r.fs.IsFileExist(filepath.Join("objects", hash))
}

func (r *FSRepo) insertObject(hash string, src FsFile) error {
	target := filepath.Join("objects", hash[:2], hash[2:])
	return r.fs.Move(src.Name(), target)
}

// scan pack indexes
func (r *FSRepo) scanPacks() error {
	if !r.fs.IsFileExist("objects/pack") {
		return nil
	}

	names, err := r.fs.ListDir("objects/pack/")
	if err != nil {
		return err
	}

	for _, name := range names {
		if strings.HasSuffix(name, ".idx") {
			// extract hash from pack name
			id := name[18 : len(name)-4]
			packFileName := name[:len(name)-4] + ".pack"

			idxFile, err := r.fs.Open(name)
			if err != nil {
				return err
			}

			packFile, err := r.fs.Open(packFileName)
			if err != nil {
				return err
			}

			pack, err := OpenPack(idxFile, packFile)
			if err != nil {
				return err
			}
			r.packs[id] = pack
		}
	}
	return nil
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
	file       FsFile
	zlibWriter *zlib.Writer
	hashWriter hash.Hash
	id         string
	io.WriteCloser
}

func newObjectWriter(file FsFile, size uint64, repo *FSRepo) *objectWriter {
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
		objectInfo.Type = OTypeBlob
	case "commit":
		objectInfo.Type = OTypeCommit
	case "tag":
		objectInfo.Type = OTypeTag
	case "tree":
		objectInfo.Type = OTypeTree
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

func readRefFile(fs Fs, path string) (string, error) {
	file, err := fs.Open(path)
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

func writeRefFile(fs Fs, path string, data string) error {
	file, err := fs.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write([]byte(data))

	return err
}
