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

type CommitTraceMap map[rawgit.OID]*CommitTraceItem

func NewCommitTraceMap() CommitTraceMap {
	return make(CommitTraceMap)
}

func (cs CommitTraceMap) Add(item *CommitTraceItem) {
	cs[*item.GetOID()] = item
}

func (cs CommitTraceMap) Has(item *CommitTraceItem) bool {
	_, ok := cs[*item.GetOID()]
	return ok
}

func (cs CommitTraceMap) Delete(item *CommitTraceItem) {
	delete(cs, *item.GetOID())
}

func (cs CommitTraceMap) Get(commit *rawgit.Commit) *CommitTraceItem {
	trace := cs[*commit.GetOID()]
	if trace == nil {
		trace = &CommitTraceItem{Commit: commit}
		cs[*commit.GetOID()] = trace
	}

	return trace
}
