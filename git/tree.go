package git

import (
    "fmt"
    "io"
    "strconv"
)


const TREE_ENTRY_BUFFER = 1024


type Tree struct {
    items []TreeItem
}

type TreeItem struct {
    name string
    mode uint32
    hash string
}

func ReadTree(obj io.ReadCloser) (*Tree, error) {
    var tree = new(Tree)
    var hashbuf = make([]byte, 20)
    var infobuf = make([]byte, TREE_ENTRY_BUFFER)

    defer obj.Close()

    for {
        var item TreeItem
        // read mode
        buf, err := scanUntil(obj, ' ', infobuf)
        if err != nil {
            if err != io.EOF {
                return nil, err
            } else {
                break
            }
        }

        mode, err := strconv.ParseUint(string(buf), 8, 32)
        if err != nil {
            return nil, err
        }

        item.mode = uint32(mode)

        // read name
        buf, err = scanUntil(obj, 0, infobuf)
        if err != nil {
            return nil, err
        }

        item.name = string(buf)

        // read hash

        _, err = obj.Read(hashbuf)
        if err != nil {
            return nil, err
        }

        item.hash = fmt.Sprintf("%x", hashbuf)
        tree.items = append(tree.items, item)
    }
    return tree, nil
}
