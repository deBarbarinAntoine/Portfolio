package data

import (
	"database/sql"
	"errors"
	"html/template"
	"time"
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
	TokenModel *TokenModel
	UserModel  *UserModel
	PostModel  *PostModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		TokenModel: &TokenModel{db},
		UserModel:  &UserModel{db},
		PostModel:  &PostModel{db},
	}
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

type envelope map[string]any

type Token struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

type Post struct {
	ID         int           `json:"id"`
	Title      string        `json:"title"`
	Images     []string      `json:"images"`
	Content    template.HTML `json:"content"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Popularity int           `json:"popularity,omitempty"`
	Version    int           `json:"version,omitempty"`
}
