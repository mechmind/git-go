package git

import (
    "compress/zlib"
    "errors"
    "io"
    "os"
    "path/filepath"
    "strconv"
)


const HEADER_BUFFER = 20

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
    println(r.getObjectPath(hash))
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

func (r *FSRepo) CreateObject(size int64) (io.WriteCloser, error) {
    return nil, nil
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
