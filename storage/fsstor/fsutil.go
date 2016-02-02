package fsstor

import (
	"path/filepath"
	"sort"
	"strings"
)

func Glob(fs FS, pattern string) (matches []string, err error) {
	if globber, ok := fs.(Globber); ok {
		return globber.Glob(pattern)
	}

	if !hasMeta(pattern) {
		if !fs.IsFileExist(pattern) {
			return nil, nil
		}
		return []string{pattern}, nil
	}

	dir, file := filepath.Split(pattern)
	switch dir {
	case "":
		dir = "."
	case string(filepath.Separator):
		// nothing
	default:
		dir = dir[0 : len(dir)-1] // chop off trailing separator
	}

	if !hasMeta(dir) {
		return glob(fs, dir, file, nil)
	}

	var m []string
	m, err = Glob(fs, dir)
	if err != nil {
		return
	}
	for _, d := range m {
		matches, err = glob(fs, d, file, matches)
		if err != nil {
			return
		}
	}
	return
}

// glob searches for files matching pattern in the directory dir
// and appends them to matches. If the directory cannot be
// opened, it returns the existing matches. New matches are
// added in lexicographical order.
func glob(fs FS, dir, pattern string, matches []string) (m []string, e error) {
	m = matches
	if !fs.IsFileExist(dir) {
		return
	}
	if !fs.IsDir(dir) {
		return
	}
	names, err := fs.ListDir(dir)
	if err != nil {
		return
	}

	sort.Strings(names)

	for _, n := range names {
		matched, err := filepath.Match(pattern, n)
		if err != nil {
			return m, err
		}
		if matched {
			m = append(m, filepath.Join(dir, n))
		}
	}
	return
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
func hasMeta(path string) bool {
	// TODO(niemeyer): Should other magic characters be added here?
	return strings.IndexAny(path, "*?[") >= 0
}
