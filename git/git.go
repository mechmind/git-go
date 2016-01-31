package git

import (
	"github.com/mechmind/git-go/rawgit"
	"github.com/mechmind/git-go/storage/fsstor"
)

func OpenRepository(path string) (*Repository, error) {
	storage, err := fsstor.OpenFSStorage(fsstor.NewOSFS(path))
	if err != nil {
		return nil, err
	}

	return NewRepository(rawgit.NewRepository(storage, storage)), nil
}
