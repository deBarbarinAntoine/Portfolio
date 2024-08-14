package data

import (
	"database/sql"
	"errors"
)

const (
	UserToActivate = "to-activate"
	UserActivated  = "activated"

	TokenActivation = "activation"
	TokenReset      = "reset"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	TokenModel  *TokenModel
	UserModel   *UserModel
	PostModel   *PostModel
	AuthorModel *AuthorModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		TokenModel:  &TokenModel{db},
		UserModel:   &UserModel{db},
		PostModel:   &PostModel{db},
		AuthorModel: &AuthorModel{db},
	}
}
