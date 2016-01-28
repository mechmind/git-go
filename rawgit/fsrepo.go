package rawgit

import (
	"compress/zlib"
	"crypto/sha1"
	"hash"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

const HEADER_BUFFER = 30

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

func (r *FSRepo) OpenObject(oid *OID) (ObjectInfo, io.ReadCloser, error) {
	hash := oid.String()
	path := filepath.Join("objects", hash[:2], hash[2:])
	if !r.fs.IsFileExist(path) {
		// lookup object in packs
		for _, pack := range r.packs {
			if pack.HasObject(oid) {
				return pack.OpenObject(oid)
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

	objectInfo.OID = *oid

	return objectInfo, newObjectReader(reader, objectInfo.Size), nil
}

func (r *FSRepo) StatObject(oid *OID) (ObjectInfo, interface{}, error) {
	return ObjectInfo{}, nil, nil
}

func (r *FSRepo) CreateObject(objType OType, size uint64) (ObjectWriter, error) {
	if objType <= OTypeNone || objType >= OTypeUnset {
		return nil, ErrInvalidObjectType
	}
	var objTypeName = objType.String()

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

func (r *FSRepo) IsObjectExists(oid *OID) bool {
	hash := oid.String()
	target := filepath.Join("objects", hash[:2], hash[2:])
	return r.fs.IsFileExist(target)
}

func (r *FSRepo) insertObject(oid *OID, src FsFile) error {
	hash := oid.String()
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
	id         *OID
	io.WriteCloser
}

func newObjectWriter(file FsFile, size uint64, repo *FSRepo) *objectWriter {
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

	oid := OID{}
	copy(oid[:], ob.hashWriter.Sum(nil))
	ob.id = &oid

	// insert object into repo storage
	return ob.repo.insertObject(ob.id, ob.file)
}

func (ob *objectWriter) OID() *OID {
	return ob.id.GetOID()
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
		objectInfo.OType = OTypeBlob
	case "commit":
		objectInfo.OType = OTypeCommit
	case "tag":
		objectInfo.OType = OTypeTag
	case "tree":
		objectInfo.OType = OTypeTree
	default:
		return objectInfo, ErrInvalidObjectType
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
	return nil, ErrBufferDepleted
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
