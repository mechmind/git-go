package rawgit

import (
	"io"
	"strings"
)

type ObjectWriter interface {
	io.WriteCloser
	GetOID() *OID
}

// Repo functions:
// * raw object access (read, write, lookup)
// * ref resolving
// * history lookup?
type Repository struct {
	Storage
}

func NewRepository(storage Storage) *Repository {
	return &Repository{storage}
}

func (repo *Repository) ResolveBranch(branch string) (*OID, error) {
	return repo.Storage.ResolveRef("heads/" + branch)
}

func (repo *Repository) OpenCommit(oid *OID) (*Commit, error) {
	info, body, err := repo.Storage.OpenObject(oid)
	if err != nil {
		return nil, err
	}

	if info.GetOType() != OTypeCommit {
		return nil, ErrNotACommit
	}

	return ReadCommit(body)
}

func (repo *Repository) Parents(commit *Commit) ([]*Commit, error) {
	if len(commit.ParentOIDs) == 0 {
		return nil, nil
	}

	parents := make([]*Commit, len(commit.ParentOIDs))
	for idx, poid := range commit.ParentOIDs {
		parent, err := repo.OpenCommit(poid)
		if err != nil {
			return nil, err
		}

		parents[idx] = parent
	}

	return parents, nil
}

func (repo *Repository) OpenTree(oid *OID) (*Tree, error) {
	info, body, err := repo.Storage.OpenObject(oid)
	if err != nil {
		return nil, err
	}

	if info.GetOType() != OTypeTree {
		return nil, ErrNotACommit
	}

	return ReadTree(body)
}

func (repo *Repository) FindInTree(root *OID, path string) (OType, *OID, error) {
	if path == "" {
		return OTypeBad, nil, ErrInvalidPath
	}

	path = normalizePath("/" + path)
	parts := strings.Split(path, "/")[1:]

	return repo.find(root, parts)
}

func (repo *Repository) find(root *OID, parts []string) (OType, *OID, error) {
	if len(parts) == 0 {
		info, _, err := repo.StatObject(root)
		if err != nil {
			return OTypeBad, nil, err
		}

		return info.GetOType(), info.GetOID(), nil
	}

	next := parts[0]
	tree, err := repo.OpenTree(root)
	if err != nil {
		return OTypeBad, nil, err
	}

	item := tree.Find(next)
	if item == nil {
		return OTypeBad, nil, ErrNotFound
	}

	if len(parts) == 1 {
		return item.GetOType(), item.GetOID(), nil
	}

	return repo.find(item.GetOID(), parts[1:])
}

func (repo *Repository) OpenCursor(commitOID *OID, path string) (*Cursor, error) {
	commit, err := repo.OpenCommit(commitOID)
	if err != nil {
		return nil, err
	}

	cur := &Cursor{commit: commit, repo: repo}

	if path == "" {
		// do not even open tree
	} else {
		// force lookup from root
		path = normalizePath("/" + path)
		cur.path = path

		//cur.openCurrentTree()
	}

	if cur.Err() != nil {
		return nil, cur.Err()
	}

	return cur, nil
}

// TODO: tags
