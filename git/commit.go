package git

import (
	"errors"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

const COMMIT_ENTRY_BUFFER = 1024

var (
	ErrInvalidHashLength = errors.New("invalid hash length")
	ErrNoTree            = errors.New("no proper 'tree' record in commit object")
	ErrNoAuthor          = errors.New("no proper 'author' record in commit object")
	ErrNoCommitter       = errors.New("no proper 'commiter' record in commit object")
	ErrInvalidEmail      = errors.New("no proper email field in record")
	ErrInvalidEncoding   = errors.New("malformed encoding record")
	ErrInvalidRecord     = errors.New("invalid record")
)

type UserTime struct {
	Name, Email string
	Time        time.Time
}

type Commit struct {
	TreeId    string
	ParentIds []string
	Author    UserTime
	Committer UserTime
	Encoding  string
	Message   string
}

func ReadCommit(obj io.ReadCloser) (*Commit, error) {
	var commit = new(Commit)

	var infobuf = make([]byte, COMMIT_ENTRY_BUFFER)
	var name, hash string

	defer obj.Close()

	// read tree id
	buf, err := scanUntil(obj, ' ', infobuf)
	if err != nil {
		return nil, err
	}
	name = string(buf)
	if name != "tree" {
		return nil, ErrNoTree
	}

	buf, err = scanUntil(obj, '\n', infobuf)
	if err != nil {
		return nil, err
	}

	hash = string(buf)
	if len(hash) != 40 {
		return nil, ErrInvalidHashLength
	}
	commit.TreeId = hash

	// read parents, if any
	for {
		buf, err = scanUntil(obj, ' ', infobuf)
		if err != nil {
			return nil, err
		}
		name = string(buf)
		if name != "parent" {
			// end of parents
			break
		}

		buf, err = scanUntil(obj, '\n', infobuf)
		if err != nil {
			return nil, err
		}
		hash = string(buf)
		if len(hash) != 40 {
			return nil, ErrInvalidHashLength
		}
		commit.ParentIds = append(commit.ParentIds, hash)
	}
	// read author record
	// 'name' already read
	if name != "author" {
		return nil, ErrNoAuthor
	}

	userTime, err := readUserRecord(obj, infobuf)
	if err != nil {
		return nil, err
	}

	commit.Author = userTime

	// read commiter record
	buf, err = scanUntil(obj, ' ', infobuf)
	if err != nil {
		return nil, err
	}

	name = string(buf)
	if name != "committer" {
		return nil, ErrNoCommitter
	}

	userTime, err = readUserRecord(obj, infobuf)
	if err != nil {
		return nil, err
	}

	commit.Committer = userTime

	// read encoding, if any
	buf, err = scanUntil(obj, '\n', infobuf)
	if err != nil {
		return nil, err
	}

	encLine := string(buf)
	if encLine != "" {
		idx := strings.Index(encLine, " ")
		if idx == -1 {
			return nil, ErrInvalidEncoding
		}
		if encLine[:idx] != "encoding" {
			return nil, ErrInvalidEncoding
		}
		commit.Encoding = encLine[idx+1:]

		// read empty line before message
		buf, err = scanUntil(obj, '\n', infobuf)
		if err != nil {
			return nil, err
		}
		if len(buf) != 0 {
			return nil, ErrInvalidRecord
		}
	}

	// read commit message
	msg, err := ioutil.ReadAll(obj)
	if err != nil {
		return nil, err
	}

	commit.Message = string(msg)

	return commit, nil
}

func readUserRecord(obj io.ReadCloser, infobuf []byte) (UserTime, error) {
	buf, err := scanUntil(obj, '<', infobuf)
	if err != nil {
		return UserTime{}, err
	}

	if len(buf) < 1 {
		return UserTime{}, ErrNoAuthor
	}
	username := string(buf[:len(buf)-1])

	buf, err = scanUntil(obj, ' ', infobuf)
	if err != nil {
		return UserTime{}, err
	}

	if len(buf) < 1 {
		return UserTime{}, ErrInvalidEmail
	}
	email := string(buf[:len(buf)-1])

	buf, err = scanUntil(obj, ' ', infobuf)
	if err != nil {
		return UserTime{}, err
	}
	timestamp, err := strconv.ParseInt(string(buf), 10, 32)

	if err != nil {
		return UserTime{}, err
	}

	buf, err = scanUntil(obj, '\n', infobuf)
	if err != nil {
		return UserTime{}, err
	}

	timezone, err := strconv.ParseInt(string(buf), 10, 32)
	if err != nil {
		return UserTime{}, err
	}
	timezone /= 100

	// git does not provide timezone info, so we use fake timezone with proper time shift
	location := time.FixedZone("GIT", int(timezone)*60*60)
	commitDate := time.Unix(timestamp, 0).In(location)

	userTime := UserTime{username, email, commitDate}
	return userTime, nil
}
