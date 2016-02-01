package rawgit

import (
	"io"
	"io/ioutil"
)

type Tag struct {
	OID
	OType
	TargetOID   OID
	TargetOType OType
	Name        string
	Tagger      UserTime
	Message     string
}

func ReadTag(obj io.ReadCloser) (*Tag, error) {
	var tag = new(Tag)
	var infobuf = make([]byte, 1024)

	tag.OType = OTypeTag

	defer obj.Close()

	buf, err := scanUntil(obj, ' ', infobuf)
	if string(buf) != "object" {
		return nil, ErrNoObject
	}

	buf, err = scanUntil(obj, '\n', infobuf)

	oid, err := ParseOID(string(buf))
	if err != nil {
		return nil, err
	}

	tag.TargetOID = *oid

	buf, err = scanUntil(obj, ' ', infobuf)
	if string(buf) != "type" {
		return nil, ErrNoObjectType
	}

	buf, err = scanUntil(obj, '\n', infobuf)
	ot := ParseOType(string(buf))
	if ot == OTypeBad {
		return nil, ErrInvalidObjectType
	}
	tag.TargetOType = ot
	println("parsed otype", ot.String())

	buf, err = scanUntil(obj, ' ', infobuf)
	if string(buf) != "tag" {
		return nil, ErrNoTag
	}

	buf, err = scanUntil(obj, '\n', infobuf)
	tag.Name = string(buf)

	buf, err = scanUntil(obj, ' ', infobuf)
	if string(buf) != "tagger" {
		return nil, ErrNoTagger
	}

	userTime, err := readUserRecord(obj, infobuf)
	if err != nil {
		return nil, err
	}
	tag.Tagger = userTime

	// scan empty line before message
	buf, err = scanUntil(obj, '\n', infobuf)
	if err != nil {
		return nil, err
	} else if len(buf) > 0 {
		return nil, ErrInvalidRecord
	}

	buf, err = ioutil.ReadAll(obj)
	if err != nil {
		return nil, err
	}

	tag.Message = string(buf)

	return tag, nil
}
