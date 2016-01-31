package rawgit

import (
	"io"
	"strconv"
)

const (
	TreeEntryBufferSize    = 1024
	TreeDirectoryMode      = 040000
	TreeBlobMode           = 0100644
	TreeExecutableBlobMode = 0100755
	TreeSymlinkMode        = 0120000
	TreeCommitMode         = 0160000
)

type Tree struct {
	Items []TreeItem
}

func (tree *Tree) Find(name string) *TreeItem {
	for idx := range tree.Items {
		if tree.Items[idx].Name == name {
			return &tree.Items[idx]
		}
	}
	return nil
}

type TreeItem struct {
	Name string
	Mode uint32
	OID  OID
}

func (item *TreeItem) GetOType() OType {
	if item.Mode&TreeDirectoryMode > 0 {
		return OTypeTree
	} else {
		return OTypeBlob
	}
}

func (item *TreeItem) GetOID() *OID {
	return &item.OID
}

func ReadTree(obj io.ReadCloser) (*Tree, error) {
	var tree = new(Tree)
	var hashbuf = make([]byte, 20)
	var infobuf = make([]byte, TreeEntryBufferSize)

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

		oid, err := OIDFromBytes(hashbuf)
		if err != nil {
			return nil, err
		}

		item.OID = *oid
		tree.Items = append(tree.Items, item)
	}

	return tree, nil
}
