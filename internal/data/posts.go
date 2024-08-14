package data

import (
	"Portfolio/internal/validator"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"time"
)

var (
	ErrDuplicatePostTitle = errors.New("duplicate post title")
)

type Post struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Images    []string  `json:"images"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Views     int       `json:"views,omitempty"`
	Version   int       `json:"version,omitempty"`
}

func (post *Post) Validate(v *validator.Validator) {
	v.Check(len(post.Content) > 2, "content", "must be at least 2 bytes long")
	v.Check(len(post.Content) < 1_020, "content", "must not be more than 1.020 bytes long")
	v.StringCheck(post.Title, 2, 125, true, "title")
	v.Check(len(post.Images) > 1, "images", "must contain at least 1 image")
}

type PostModel struct {
	db *sql.DB
}

func (m PostModel) Insert(post *Post) error {

	// generating the query
	query := `
		INSERT INTO posts (title, images, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, version;`

	// setting the arguments
	args := []any{post.Content, pq.Array(post.Images), post.Content}

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// executing the query
	err = stmt.QueryRowContext(ctx, args...).Scan(&post.ID, &post.CreatedAt, &post.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "posts_title_key"`:
			return ErrDuplicatePostTitle
		default:
			return err
		}
	}

	return nil
}

func (m PostModel) Get(search string, filters *Filters) ([]*Post, Metadata, error) {

	// generating the query
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, created_at, updated_at, title, images, content, views, version
		FROM posts
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		OR (to_tsvector('simple', content) @@ plainto_tsquery('simple', $1) OR $1 = '')
		OR (images @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4;`, filters.sortColumn(), filters.sortDirection())

	// setting the arguments
	args := []any{search, pq.Array([]string{search}), filters.limit(), filters.offset()}

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, Metadata{}, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// executing the query
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	// creating the variables
	totalRecords := 0
	var posts []*Post

	// scanning for values
	for rows.Next() {
		var post Post

		err := rows.Scan(
			&totalRecords,
			&post.ID,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Title,
			pq.Array(&post.Images),
			&post.Content,
			&post.Views,
			&post.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		// adding the post to the list of matching posts
		posts = append(posts, &post)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// getting the metadata
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return posts, metadata, nil
}

func (m PostModel) GetByID(id int) (*Post, error) {

	// generating the query
	query := `
		SELECT id, created_at, updated_at, title, images, content, views, version
		FROM posts
		WHERE id = $1;`

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// setting the post variable
	var post Post

	// executing the query
	err = stmt.QueryRowContext(ctx, id).Scan(
		&post.ID,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.Title,
		pq.Array(&post.Images),
		&post.Content,
		&post.Views,
		&post.Version,
	)

	// looking for errors
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &post, nil
}

func (m PostModel) Update(post Post) error {

	// generating the query
	query := `
		UPDATE posts 
		SET updated_at = NOW(), title = $1, images= $2, content = $3, version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING updated_at, version;`

	// setting the arguments
	args := []any{
		post.Title,
		pq.Array(post.Images),
		post.Content,
		post.ID,
		post.Version,
	}

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// executing the query
	err = stmt.QueryRowContext(ctx, args...).Scan(&post.UpdatedAt, &post.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m PostModel) IncrementViews(id int) error {

	// generating the query
	query := `
		UPDATE posts 
		SET views = views + 1
		WHERE id = $1;`

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// executing the query
	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (m PostModel) Delete(id int) error {

	// generating the query
	query := `
		DELETE FROM posts
		WHERE id = $1;`

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// executing the query
	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return err
	}

	// checking for result
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// if nothing found
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}
