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

    return objectInfo, reader, nil
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

    writer := newObjectWriter(tmpFile, r)

    // write header
    header := make([]byte, HEADER_BUFFER)
    pos := copy(header, []byte(objTypeName))
    header[pos] = ' '
    pos++
    pos += copy(header[pos:], []byte(strconv.FormatUint(size, 10)))
    header[pos] = 0
    header = header[:pos + 1]
    println(string(header))
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

type objectWriter struct {
    repo *FSRepo
    file *os.File
    zlibWriter *zlib.Writer
    hashWriter hash.Hash
    id string
    io.Writer
}

func newObjectWriter(file *os.File, repo *FSRepo) *objectWriter {
    zw := zlib.NewWriter(file)
    hw := sha1.New()
    allw := io.MultiWriter(hw, zw)
    return &objectWriter{repo, file, zw, hw, "", allw}
}

func (ob *objectWriter) closeWriters() error {
    err := ob.zlibWriter.Close()
    err2 := ob.file.Close()
    if err != nil {
        return err
    }
    return err2
}


func (ob *objectWriter) Close() error {
    ob.closeWriters()
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
