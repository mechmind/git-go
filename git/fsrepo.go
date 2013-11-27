package git

import (
    "compress/zlib"
    "crypto/sha1"
    "errors"
    "fmt"
    "hash"
    "io"
    "io/ioutil"
    "os"
    "path/filepath"
    "strconv"
)


const HEADER_BUFFER = 30

var ErrObjectNotFound = errors.New("object not found")


type FSRepo struct {
    gitDir string
}

func OpenRepo(dir string) (Repo, error) {
    gitDir := filepath.Join(dir, ".git")
    if _, err := os.Stat(gitDir); err != nil {
        return nil, err
    }

    return &FSRepo{gitDir}, nil
}

func (r *FSRepo) OpenObject(hash string) (ObjectInfo, io.ReadCloser, error) {
    // only opens loose objects now
    if ! r.IsObjectExists(hash) {
        return ObjectInfo{}, nil, ErrObjectNotFound
    }

    filePath := r.getObjectPath(hash)
    file, err := os.Open(filePath)
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

    tempDir := filepath.Join(r.gitDir, "objects", "temp")
    if _, err := os.Stat(tempDir); os.IsNotExist(err) {
        if err = os.MkdirAll(tempDir, 0755); err != nil {
            return nil, err
        }
    }

    tmpFile, err := ioutil.TempFile(tempDir, "object")
    if err != nil {
        return nil, err
    }

    // create header
    header := make([]byte, HEADER_BUFFER)
    pos := copy(header, []byte(objTypeName))
    header[pos] = ' '
    pos++
    pos += copy(header[pos:], []byte(strconv.FormatUint(size, 10)))
    header[pos] = 0
    header = header[:pos + 1]

    writer := newObjectWriter(tmpFile, size + uint64(len(header)), r)

    if _, err = writer.Write(header); err != nil {
        writer.closeWriters()
        os.Remove(tmpFile.Name())
        return nil, err
    }

    return writer, nil
}


func (r *FSRepo) IsObjectExists(hash string) bool {
    if _, err := os.Stat(r.getObjectPath(hash)); err != nil {
        return false
    }
    return true
}


func (r *FSRepo) getObjectPath(hash string) string {
    if len(hash) < 2 {
        return ""
    }
    return filepath.Join(r.gitDir, "objects", hash[:2], hash[2:])
}

func (r *FSRepo) insertObject(hash string, fileName string) error {
    dir := filepath.Join(r.gitDir, "objects", hash[:2])
    objectFileName := filepath.Join(dir, hash[2:])
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        if err = os.Mkdir(dir, 0755); err != nil {
            return err
        }
    }

    return os.Rename(fileName, objectFileName)
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
    repo *FSRepo
    file *os.File
    zlibWriter *zlib.Writer
    hashWriter hash.Hash
    id string
    io.WriteCloser
}

func newObjectWriter(file *os.File, size uint64, repo *FSRepo) *objectWriter {
    zw := zlib.NewWriter(file)
    hw := sha1.New()
    allw := &exactSizeWriter{size, io.MultiWriter(hw, zw)}
    return &objectWriter{repo, file, zw, hw, "", allw}
}

func (ob *objectWriter) closeWriters() error {
    err1 := ob.WriteCloser.Close()
    err2 := ob.zlibWriter.Close()
    err3 := ob.file.Close()
    if err1 != nil {
        return err1
    }
    if err2 != nil {
        return err2
    }
    return err3
}


func (ob *objectWriter) Close() error {
    err := ob.closeWriters()
    if err != nil {
        os.Remove(ob.file.Name())
        return err
    }
    ob.id = fmt.Sprintf("%x", ob.hashWriter.Sum(nil))

    // insert object into repo storage
    return ob.repo.insertObject(ob.id, ob.file.Name())
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
    for i := 0; i < cap(buf) - 1; i++ {
        _, err := src.Read(buf[i:i + 1])
        if err != nil {
            return nil, err
        }
        if buf[i:i + 1][0] == needle {
            return buf[:i], nil
        }
    }
    return nil, errors.New("buffer depleted")
}
