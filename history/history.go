package history

import (
	"container/list"

	"github.com/mechmind/git-go/rawgit"
)

type WalkerAction int

const (
	// drop commit and do not follow parents
	DropCommit WalkerAction = 0
	// take commit but not traverse parents
	TakeCommit WalkerAction = 1 << iota
	// drop commit but traverse parents
	FollowParents
	// stop traverse
	Stop

	// take commit and follow parents
	TakeAndFollow = TakeCommit | FollowParents
)

type WalkerCallback func(*rawgit.Commit) (WalkerAction, error)

// CommitComparator defines callback type for checking commit equalty. If it returns true then
// commits are considered equal. See "History Simplification" chapter of git-log man for details
type CommitComparator func(current, parent *rawgit.Commit) bool

type History struct {
	repo *rawgit.Repository
}

func New(repo *rawgit.Repository) *History {
	return &History{repo}
}

func (hist *History) WalkHistory(start *rawgit.OID, callback WalkerCallback) (*list.List, error) {
	return hist.WalkFilteredHistory(start, callback, ExactCommitComparator)
}

func (hist *History) WalkFilteredHistory(start *rawgit.OID, callback WalkerCallback,
	eq CommitComparator) (*list.List, error) {

	commit, err := hist.repo.OpenCommit(start)
	if err != nil {
		return nil, err
	}

	if callback == nil {
		callback = NopCallback
	}

	return hist.walk([]*rawgit.Commit{commit}, callback, eq)
}

// roots must be not equal to each other
func (hist *History) walk(roots []*rawgit.Commit, callback WalkerCallback,
	eq CommitComparator) (*list.List, error) {

	results := list.New()
	seen := NewCommitSet()

	for {
		if len(roots) == 0 {
			return results, nil
		}

		var err error

		roots, err = hist.simplifyRoots(roots, eq, seen)
		if err != nil {
			return nil, err
		}

		if len(roots) == 0 {
			return results, nil
		}

		var next *rawgit.Commit
		next, roots = extractNewestCommit(roots)

		action, err := callback(next)
		if err != nil {
			return nil, err
		}

		seen.Add(next.GetOID())

		if action&TakeCommit > 0 {
			// take commit
			results.PushBack(next)
		}

		if action&FollowParents > 0 {
			// follow all parents of commit
			pars, err := hist.parents(next)
			if err != nil {
				return nil, err
			}
			roots = mergeRoots(pars, roots, eq, seen)
		}

		if action&Stop > 0 {
			return results, nil
		}
	}

	return results, nil
}

func (hist *History) parents(commit *rawgit.Commit) ([]*rawgit.Commit, error) {
	parents := make([]*rawgit.Commit, len(commit.ParentOIDs))
	for idx := 0; idx < len(parents); idx++ {
		var err error
		parents[idx], err = hist.repo.OpenCommit(commit.ParentOIDs[idx])
		if err != nil {
			return nil, err
		}
	}
	return parents, nil
}

// mergeRoots will merge two sets of commits and ensure that they are not equal to each other
// the members of base and merging sets already nonequal to each other
func mergeRoots(base, merging []*rawgit.Commit, eq CommitComparator, seen CommitSet) []*rawgit.Commit {
	newRoots := append([]*rawgit.Commit(nil), base...)
	for _, needle := range merging {
		found := false
		for _, item := range base {
			if needle.GetOID().Equal(item.GetOID()) {
				// we struck merge base!
				// drop it
				found = true
				break
			}
			if eq(needle, item) {
				// found equal commit in merging roots
				// witness and drop it
				seen.Add(needle.GetOID())
				found = true
				break
			}
		}

		if !found {
			// commit is unique enough
			newRoots = append(newRoots, needle)
		}
	}

	return newRoots
}

// skipEqualCommits compares commit to it parents. If it finds a parent
// that equals to current commit the current commit will be dropped and parent will be followed
// see "History Simplification" chapter of git-log man for full details.
func (hist *History) skipEqualCommits(commit *rawgit.Commit, eq CommitComparator,
	seen CommitSet) (*rawgit.Commit, error) {

	for {
		// we already seen that commit, no point to traverse further
		if seen.Has(commit.GetOID()) {
			return nil, nil
		}

		if len(commit.ParentOIDs) == 0 {
			return commit, nil
		}

		var found bool
		for idx := 0; idx < len(commit.ParentOIDs); idx++ {
			parent, err := hist.repo.OpenCommit(commit.ParentOIDs[idx])
			if err != nil {
				return nil, err
			}

			if eq(commit, parent) {
				// we have parent that equals to given commit
				// so take this parent as next commit (dropping current)
				// but we will remember that we seen current commit in history
				seen[*commit.GetOID()] = struct{}{}
				commit = parent
				found = true
				break
			}
		}

		// all parents of commit was not same to it, so return this commit
		if !found {
			return commit, nil
		}
	}
}

func (hist *History) simplifyRoots(roots []*rawgit.Commit, eq CommitComparator,
	seen map[rawgit.OID]struct{}) ([]*rawgit.Commit, error) {

	newRoots := []*rawgit.Commit{}
	for _, commit := range roots {
		commit, err := hist.skipEqualCommits(commit, eq, seen)
		if err != nil {
			return nil, err
		}
		if commit != nil {
			newRoots = append(newRoots, commit)
		}
	}

	return newRoots, nil
}

// extractNewestCommit will find newest commit, extract it and return resulting set
func extractNewestCommit(roots []*rawgit.Commit) (*rawgit.Commit, []*rawgit.Commit) {
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
