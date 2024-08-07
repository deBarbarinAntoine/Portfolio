package data

import (
	"database/sql"
	"html/template"
	"time"
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

type User struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Avatar    string    `json:"avatar,omitempty"`
	Status    string    `json:"status"`
	Version   int       `json:"-"`
}

type Post struct {
	ID         int           `json:"id"`
	Images     []string      `json:"images"`
	Content    template.HTML `json:"content"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Popularity int           `json:"popularity,omitempty"`
	Version    int           `json:"version,omitempty"`
}
