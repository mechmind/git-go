package history

import (
	"regexp"

	"github.com/mechmind/git-go/rawgit"
)

func CommitRootComparator(current, parent *rawgit.Commit) bool {
	return current.TreeOID.Equal(parent.TreeOID)
}

func NopComparator(current, parent *rawgit.Commit) bool {
	return false
}

func ExactCommitComparator(current, parent *rawgit.Commit) bool {
	return current.GetOID().Equal(parent.GetOID())
}

func MakePathComparator(repo rawgit.Repository, path string) CommitComparator {
	return func(current, parent *rawgit.Commit) bool {
		_, centry, cerr := repo.FindInTree(current.TreeOID, path)
		_, pentry, perr := repo.FindInTree(parent.TreeOID, path)

		if cerr != nil || perr != nil {
			return cerr == ErrNotFound && perr == ErrNotFound
		}

		return centry.Equal(pentry)
	}
}

func MakePathChecker(repo rawgit.Repository, path string) (cb WalkerCallback) {
	return func(commit *rawgit.Commit) (WalkerAction, error) {
		_, _, err := repo.FindInTree(commit.TreeOID, path)
		if err != nil {
			if err == rawgit.ErrNotFound {
				return FollowParents, nil
			} else {
				return Stop, err
			}
		}

		return TakeAndFollow, nil
	}
}

func MakeHistorySearcher(needle string) (cb WalkerCallback, err error) {
	matcher, err := regexp.Compile(needle)
	if err != nil {
		return nil, err
	}

	return func(commit *rawgit.Commit) (WalkerAction, error) {
		if matcher.MatchString(commit.Message) {
			return TakeAndFollow, nil
		}

		return FollowParents, nil
	}, nil
}

func NopCallback(*rawgit.Commit) (WalkerAction, error) {
	return TakeAndFollow, nil
}

func MakePager(repo rawgit.Repository, cb WalkerCallback, skip int, count int) WalkerCallback {
	if cb == nil {
		cb = NopCallback
	}

	pagerCallback := func(commit *rawgit.Commit) (WalkerAction, error) {
		pagerAction, err := cb(commit)
		if err != nil {
			return pagerAction, err
		}

		// if checker does not want to pick this commit, pager does not want either
		if pagerAction&TakeCommit == 0 {
			return pagerAction, nil
		}

		if skip != 0 {
			skip--
			action := pagerAction &^ TakeCommit
			return action, nil
		}

		if count != 0 {
			count--
			// this is last element we want to take
			if count == 0 {
				pagerAction |= Stop
			}
			return pagerAction, nil
		}

		return Stop, nil
	}

	return pagerCallback
}

func MakeCounter(cb WalkerCallback) (WalkerCallback, func() int) {
	count := 0

	if cb == nil {
		cb = NopCallback
	}

	callback := func(commit *rawgit.Commit) (WalkerAction, error) {
		action, err := cb(commit)
		if err != nil {
			return action, err
		}

		if action&TakeCommit > 0 {
			count++
			action = action &^ TakeCommit
		}

		return action, nil
	}

	getter := func() int {
		return count
	}

	return callback, getter
}
