package history

import "github.com/mechmind/git-go/rawgit"

type CommitSet map[rawgit.OID]struct{}

func NewCommitSet() CommitSet {
	return make(CommitSet)
}

var empty = struct{}{}

func (cs CommitSet) Add(oid *rawgit.OID) {
	cs[*oid] = empty
}

func (cs CommitSet) Has(oid *rawgit.OID) bool {
	_, ok := cs[*oid]
	return ok
}
