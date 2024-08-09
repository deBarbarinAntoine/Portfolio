package data

import (
	"database/sql"
	"net/url"
)

type PostModel struct {
	db *sql.DB
}

func (m *PostModel) Create(post *Post) error {

	// TODO

	return nil
}

func (m *PostModel) Update(post *Post) error {

	// TODO

	return nil
}

func (m *PostModel) Delete(id int) error {

	// TODO

	return nil
}

func (m *PostModel) Get(query url.Values) ([]Post, Metadata, error) {

	// TODO

	var posts []Post
	var metadata Metadata

	return posts, metadata, nil
}

func (m *PostModel) GetByID(id int) (Post, error) {

	// TODO

	var post Post

	return post, nil
}
