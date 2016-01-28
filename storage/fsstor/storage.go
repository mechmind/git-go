package fsstor

import (
	"compress/zlib"
	"crypto/sha1"
	"hash"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mechmind/git-go/rawgit"
)

const HeaderBufferSize = 30

type FSStorage struct {
	fs    FS
	packs map[string]*Pack
}

func OpenFSStorage(fs FS) (*FSStorage, error) {
	repo := &FSStorage{fs, make(map[string]*Pack)}
	// load packs
	err := repo.scanPacks()
	return repo, err
}

func (r *FSStorage) OpenObject(oid *rawgit.OID) (rawgit.ObjectInfo, io.ReadCloser, error) {
	hash := oid.String()
	path := filepath.Join("objects", hash[:2], hash[2:])
	if !r.fs.IsFileExist(path) {
		// lookup object in packs
		for _, pack := range r.packs {
			if pack.HasObject(oid) {
				return pack.OpenObject(oid)
			}
		}
		return rawgit.ObjectInfo{}, nil, ErrObjectNotFound
	}

	file, err := r.fs.Open(path)
	if err != nil {
		return rawgit.ObjectInfo{}, nil, err
	}

	reader, err := zlib.NewReader(file)
	if err != nil {
		return rawgit.ObjectInfo{}, nil, err
	}

	objectInfo, err := readHeader(reader)
	if err != nil {
		reader.Close()
		return rawgit.ObjectInfo{}, nil, err
	}

	objectInfo.OID = *oid

	return objectInfo, newObjectReader(reader, objectInfo.Size), nil
}

func (r *FSStorage) StatObject(oid *rawgit.OID) (rawgit.ObjectInfo, interface{}, error) {
	return rawgit.ObjectInfo{}, nil, nil
}

func (r *FSStorage) CreateObject(objType rawgit.OType, size uint64) (rawgit.ObjectWriter, error) {
	if objType <= rawgit.OTypeNone || objType >= rawgit.OTypeUnset {
		return nil, ErrInvalidObjectType
	}
	var objTypeName = objType.String()

	// create header
	header := make([]byte, HeaderBufferSize)
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

func (r *FSStorage) ReadRef(ref string) (string, error) {
	// read refs till object found
	return readRefFile(r.fs, path.Join("refs", ref))
}

func (r *FSStorage) ReadSpecialRef(ref rawgit.SpecialRef) (string, error) {
	return readRefFile(r.fs, ref.String())
}

func (r *FSStorage) WriteRef(ref, value string) error {
	return writeRefFile(r.fs, path.Join("refs", ref), value)
}

func (r *FSStorage) WriteSpecialRef(ref rawgit.SpecialRef, value string) error {
	return writeRefFile(r.fs, ref.String(), value)
}

func (r *FSStorage) ListRefs(ns string) ([]string, error) {
	return r.fs.ListDir(filepath.Join("refs", ns))
}

func (r *FSStorage) IsObjectExist(oid *rawgit.OID) bool {
	hash := oid.String()
	target := filepath.Join("objects", hash[:2], hash[2:])
	return r.fs.IsFileExist(target)
}

func (r *FSStorage) IsReadOnly() bool {
	return r.fs.IsReadOnly()
}

func (r *FSStorage) insertObject(oid *rawgit.OID, src File) error {
	hash := oid.String()
	target := filepath.Join("objects", hash[:2], hash[2:])
	return r.fs.Move(src.Name(), target)
}

// scan pack indexes
func (r *FSStorage) scanPacks() error {
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
	repo       *FSStorage
	file       File
	zlibWriter *zlib.Writer
	hashWriter hash.Hash
	id         *rawgit.OID
	io.WriteCloser
}

func newObjectWriter(file File, size uint64, repo *FSStorage) *objectWriter {
	zw := zlib.NewWriter(file)
	hw := sha1.New()
	allw := &exactSizeWriter{size, io.MultiWriter(hw, zw)}
	return &objectWriter{repo, file, zw, hw, nil, allw}
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

	oid := rawgit.OID{}
	copy(oid[:], ob.hashWriter.Sum(nil))
	ob.id = &oid

	// insert object into repo storage
	return ob.repo.insertObject(ob.id, ob.file)
}

func (ob *objectWriter) GetOID() *rawgit.OID {
	return ob.id.GetOID()
}

func readHeader(src io.Reader) (rawgit.ObjectInfo, error) {
	var objectInfo rawgit.ObjectInfo
	var buf = make([]byte, HeaderBufferSize)

	// read type of object
	obuf, err := scanUntil(src, ' ', buf)
	if err != nil {
		return objectInfo, err
	}

	objType := string(obuf)
	switch objType {
	case "blob":
		objectInfo.OType = rawgit.OTypeBlob
	case "commit":
		objectInfo.OType = rawgit.OTypeCommit
	case "tag":
		objectInfo.OType = rawgit.OTypeTag
	case "tree":
		objectInfo.OType = rawgit.OTypeTree
	default:
		return objectInfo, ErrInvalidObjectType
	}

	// read length of object
	obuf, err = scanUntil(src, 0, buf)
	if err != nil {
		return rawgit.ObjectInfo{}, err
	}

	size, err := strconv.ParseUint(string(obuf), 10, 64)
	if err != nil {
		return rawgit.ObjectInfo{}, err
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
	return nil, ErrBufferDepleted
}

func readRefFile(fs FS, path string) (string, error) {
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

func writeRefFile(fs FS, path string, data string) error {
	file, err := fs.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write([]byte(data))

	return err
}

type exactSizeWriter struct {
	bytesLeft uint64
	writer    io.Writer
}

func (e *exactSizeWriter) Write(data []byte) (n int, err error) {
	length := uint64(len(data))
	if length == 0 {
		return 0, nil
	}

	if length > e.bytesLeft {
		// FIXME: should be instant fail
		// write remaining chunk and return overflow error
		println("want to write ", length, " bytes, but left only ", e.bytesLeft)
		data = data[:int(e.bytesLeft)]
		n, err := e.writer.Write(data)
		e.bytesLeft -= uint64(n)
		if err != nil {
			return n, err
		} else {
			return n, ErrObjectOverflow
		}
	} else {
		n, err := e.writer.Write(data)
		e.bytesLeft -= uint64(n)
		return n, err
	}
}

func (e *exactSizeWriter) Close() error {
	if e.bytesLeft != 0 {
		return ErrIncompletedObject
	}
	return nil
}
