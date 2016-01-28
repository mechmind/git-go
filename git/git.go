package git

import (
	"github.com/mechmind/git-go/rawgit"
	"github.com/mechmind/git-go/storage/fsstor"
)

func OpenRepository(path string) (*rawgit.Repository, error) {
	storage, err := fsstor.OpenFSStorage(fsstor.NewOSFS(path))
	if err != nil {
		return nil, err
	}

	return rawgit.NewRepository(storage), nil
}
