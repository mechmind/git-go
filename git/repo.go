package git

import (
	"github.com/mechmind/git-go/rawgit"
)

type Repository struct {
	rawgit.Repository
}

func NewRepository(base rawgit.Repository) *Repository {
	return &Repository{base}
}
