package rawgit

import (
	"fmt"
	"io"
	"strconv"
)

const (
	TREE_MODE_DIR     = 040000
	TREE_ENTRY_BUFFER = 1024
)

type Tree struct {
	Items []TreeItem
}

type TreeItem struct {
	Name string
	Mode uint32
	Hash string
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

		item.Mode = uint32(mode)

		// read name
		buf, err = scanUntil(obj, 0, infobuf)
		if err != nil {
			return nil, err
		}

		item.Name = string(buf)

		// read hash

		_, err = obj.Read(hashbuf)
		if err != nil {
			return nil, err
		}

		item.Hash = fmt.Sprintf("%x", hashbuf)
		tree.Items = append(tree.Items, item)
	}
	return tree, nil
}
