package rawgit

import (
	"io"
	"path"
	"strings"
)

type Cursor struct {
	commit *Commit
	tree   *Tree
	path   string
	repo   Repository

	err error
}

func (cur *Cursor) Err() error {
	return cur.err
}

func (cur *Cursor) CD(newPath string) {
	if newPath == "" {
		// "close" the tree
		cur.path = ""
		cur.tree = nil
		return
	}

	if newPath[0] == '/' {
		cur.path = normalizePath(path.Join(cur.path, newPath))
	} else {
		cur.path = normalizePath(newPath)
	}
}

func (cur *Cursor) OpenBlob(path string) (ObjectInfo, io.ReadCloser, error) {
	return ObjectInfo{}, nil, nil
}

func (cur *Cursor) openTree(targetPath string) (*Tree, error) {
	if targetPath == "" {
		return nil, ErrInvalidPath
	}

	if targetPath[0] != '/' {
		targetPath = path.Join(cur.path, targetPath)
	} else {
		targetPath = path.Clean(targetPath)
	}

	parts := strings.Split(targetPath, "/")

	tree, err := cur.repo.OpenTree(cur.commit.TreeOID)
	if err != nil {
		return nil, nil
	}

	for _, part := range parts {
		cur.tree, cur.err = cur.findSubTree(part)
		if cur.err != nil {
			return nil, nil
		}
	}

	return tree, nil
}

func (cur *Cursor) findSubTree(name string) (*Tree, error) {
	obj := cur.tree.Find(name)

	if obj.GetOType() != OTypeTree {
		return nil, ErrNotATree
	}

	return cur.repo.OpenTree(obj.GetOID())
}

func normalizePath(curPath string) string {
	return path.Clean(curPath)
}
