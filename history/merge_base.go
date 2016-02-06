package history

import (
	"github.com/mechmind/git-go/rawgit"
)

const (
	TraceP1 = 1 << iota
	TraceP2
	TraceStale
	TraceResult
)

type CommitTraceItem struct {
	*rawgit.Commit
	traceMark int
}

func (hist *History) Find3WayMergeBase(roots ...*rawgit.Commit) (*rawgit.Commit, error) {
	if len(roots) < 2 {
		return nil, ErrTooFewRoots
	}

	seen := NewCommitTraceMap()

	return hist.findMergeBase(seen, roots[0], roots[1:]...)
}

func haveNonStale(roots []*CommitTraceItem) bool {
	for _, root := range roots {
		if root.traceMark&TraceStale == 0 {
			return true
		}
	}

	return false
}

func (hist *History) findMergeBase(trace CommitTraceMap, left *rawgit.Commit, rights ...*rawgit.Commit) (*rawgit.Commit, error) {
	for _, commit := range rights {
		if left.GetOID().Equal(commit.GetOID()) {
			return left, nil
		}
	}

	traceLeft := trace.Get(left)
	traceLeft.traceMark |= TraceP1
	roots := []*CommitTraceItem{traceLeft}

	for _, commit := range rights {
		traceRight := trace.Get(commit)
		traceRight.traceMark |= TraceP2
		roots = append(roots, traceRight)
	}

	result := []*CommitTraceItem{}

	// traverse commit tree
	// implementation based on git's commit.c:paint_down_to_common
	for haveNonStale(roots) {
		var current *CommitTraceItem
		current, roots = extractNewestTraceItem(roots)

		flags := current.traceMark & (TraceP1 | TraceP2 | TraceStale)
		if flags == TraceP1|TraceP2 {
			if current.traceMark&TraceResult == 0 {
				current.traceMark |= TraceResult
				result = append(result, current)
			}
			flags |= TraceStale
		}

		parents := current.ParentOIDs
		for _, oid := range parents {
			commit, err := hist.repo.OpenCommit(oid)
			if err != nil {
				return nil, err
			}

			ptrace := trace.Get(commit)
			if ptrace.traceMark&flags == flags {
				continue
			}

			ptrace.traceMark |= flags
			roots = append(roots, ptrace)
		}
	}

	if len(result) == 0 {
		return nil, nil
	}

	newest, _ := extractNewestTraceItem(result)
	return newest.Commit, nil
}

func extractNewestTraceItem(roots []*CommitTraceItem) (*CommitTraceItem, []*CommitTraceItem) {
	if len(roots) == 1 {
		return roots[0], roots[:0]
	}

	target := roots[0]
	targetIdx := 0
	for idx, current := range roots[1:] {
		if current.Committer.Time.After(target.Committer.Time) {
			target = current
			targetIdx = idx + 1
		}
	}

	// remove picked commit
	roots = append(roots[:targetIdx], roots[targetIdx+1:]...)

	return target, roots
}
