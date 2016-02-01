package rawgit

import (
	"io"
	"strings"
)

const (
	RefPrefix   = "ref: "
	RefBranchNS = "refs/heads/"
)

type ObjectWriter interface {
	io.WriteCloser
	GetOID() *OID
}

type Repository interface {
	Storage
	RefDatabase
	ReadOnly

	ResolveBranch(branch string) (*OID, error)
	ResolveRef(ref string) (*OID, error)
	ResolveTag(ref string) (*OID, OType, error)

	OpenCommit(oid *OID) (*Commit, error)
	OpenTree(oid *OID) (*Tree, error)
	OpenTag(oid *OID) (*Tag, error)

	Parents(commit *Commit) ([]*Commit, error)
	FindInTree(oid *OID, path string) (OType, *OID, error)
}

type SimpleRepository struct {
	Storage
	refdb RefDatabase
}

func NewRepository(storage Storage, refdb RefDatabase) *SimpleRepository {
	return &SimpleRepository{storage, refdb}
}

func (repo *SimpleRepository) ReadRef(ref string) (string, error) {
	// TODO: validate ref name. Allow only refs from 'refs/' directory and special refs
	return repo.refdb.ReadRef(ref)
}

func (repo *SimpleRepository) WriteRef(ref, value string) error {
	// TODO: validate ref name. Allow only refs from 'refs/' directory and special refs
	// also validate ref value. It must be either object hash or another ref
	return repo.refdb.WriteRef(ref, value)

}

func (repo *SimpleRepository) ListRefs(ns string) ([]string, error) {
	return repo.refdb.ListRefs(ns)
}

func (repo *SimpleRepository) ResolveBranch(branch string) (*OID, error) {
	return repo.ResolveRef("refs/heads/" + branch)
}

func (repo *SimpleRepository) OpenCommit(oid *OID) (*Commit, error) {
	info, body, err := repo.Storage.OpenObject(oid)
	if err != nil {
		return nil, err
	}

	if info.GetOType() != OTypeCommit {
		return nil, ErrNotACommit
	}

	commit, err := ReadCommit(body)
	if err != nil {
		return nil, err
	}

	commit.OID = *oid
	return commit, nil
}

func (repo *SimpleRepository) Parents(commit *Commit) ([]*Commit, error) {
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

func (repo *SimpleRepository) OpenTree(oid *OID) (*Tree, error) {
	info, body, err := repo.Storage.OpenObject(oid)
	if err != nil {
		return nil, err
	}

	if info.GetOType() != OTypeTree {
		return nil, ErrNotACommit
	}

	return ReadTree(body)
}

func (repo *SimpleRepository) FindInTree(root *OID, path string) (OType, *OID, error) {
	if path == "" {
		return OTypeBad, nil, ErrInvalidPath
	}

	path = normalizePath("/" + path)
	parts := strings.Split(path, "/")[1:]

	return repo.find(root, parts)
}

func (repo *SimpleRepository) find(root *OID, parts []string) (OType, *OID, error) {
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

func (repo *SimpleRepository) OpenTag(oid *OID) (*Tag, error) {
	info, body, err := repo.OpenObject(oid)
	if err != nil {
		return nil, err
	}

	if info.GetOType() != OTypeTag {
		return nil, ErrNotATag
	}

	tag, err := ReadTag(body)
	if err != nil {
		return nil, err
	}

	tag.OID = *oid
	return tag, nil
}

func (repo *SimpleRepository) OpenCursor(commitOID *OID, path string) (*Cursor, error) {
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

func (repo *SimpleRepository) ResolveRef(ref string) (*OID, error) {
	for {
		value, err := repo.refdb.ReadRef(ref)
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(value, "ref: ") {
			ref = value[5:]
		} else {
			return ParseOID(value)
		}
	}
}

func (repo *SimpleRepository) ResolveTag(ref string) (*OID, OType, error) {
	oid, err := repo.ResolveRef("refs/tags/" + ref)
	if err != nil {
		return nil, OTypeBad, err
	}

	info, _, err := repo.StatObject(oid)
	if err != nil {
		return nil, OTypeBad, err
	}

	if info.GetOType() == OTypeTag {
		return FollowTag(repo, oid)
	}

	return oid, info.GetOType(), nil
}

func FollowTag(repo Repository, oid *OID) (*OID, OType, error) {
	for {
		tag, err := repo.OpenTag(oid)
		if err != nil {
			return nil, OTypeBad, err
		}

		oid = &tag.TargetOID
		otype := tag.TargetOType
		if otype == OTypeTag {
			// follow this tag
			continue
		}

		return oid, otype, nil
	}
}

func (repo *SimpleRepository) IsReadOnly() bool {
	// FIXME: query storage and db
	return false
}
