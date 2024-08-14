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

type Author struct {
	ID             int       `form:"id"`
	CreatedAt      time.Time `form:"-"`
	UpdatedAt      time.Time `form:"updated_at" time_format:"2006-01-02"`
	Name           string    `form:"name"`
	Email          string    `form:"email"`
	Avatar         string    `form:"avatar"`
	Birth          string    `form:"birth"`
	Location       string    `form:"location"`
	StatusActivity string    `form:"status_activity"`
	Tags           []string  `form:"tags"`
	CVFile         string    `form:"cv_file"`
	Version        int       `form:"-"`
}

func (author *Author) Validate(v *validator.Validator) {
	v.Check(author.ID == 1, "id", "invalid ID") // To assure there is only one author
	v.StringCheck(author.Name, 2, 70, true, "name")
	v.StringCheck(author.Email, 2, 120, true, "email")
	v.StringCheck(author.Avatar, 2, 250, true, "avatar")
	v.ValidateDate(author.Birth, "birth")
	v.StringCheck(author.Location, 2, 120, true, "location")
	v.StringCheck(author.StatusActivity, 2, 120, true, "status_activity")
	v.Check(validator.Unique(author.Tags), "tags", "duplicate tag")
	v.Check(len(author.Tags) < 6, "tags", "must not be more than 5")
	v.StringCheck(author.CVFile, 2, 250, true, "cv_file")
}

type AuthorModel struct {
	db *sql.DB
}

func (m AuthorModel) Get() (*Author, error) {

	// generating the query
	query := `
		SELECT id, created_at, updated_at, name, email, avatar, birth, location, status_activity, tags, cv_file, version
		FROM author
		WHERE id = 1;`

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
	var author Author

	// executing the query
	err = stmt.QueryRowContext(ctx).Scan(
		&author.ID,
		&author.CreatedAt,
		&author.UpdatedAt,
		&author.Name,
		&author.Email,
		&author.Avatar,
		&author.Birth,
		&author.Location,
		&author.StatusActivity,
		pq.Array(&author.Tags),
		&author.CVFile,
		&author.Version,
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

	return &author, nil
}

func (m AuthorModel) Update(author *Author) error {

	// generating the query
	query := `
		UPDATE author 
		SET updated_at = NOW(), name = $1, email= $2, avatar = $3, birth = $4, location = $5, status_activity = $6, tags = $7, cv_file = $8, version = version + 1
		WHERE id = 1 AND version = $9
		RETURNING updated_at, version;`

	// setting the arguments
	args := []any{
		&author.Name,
		&author.Email,
		&author.Avatar,
		&author.Birth,
		&author.Location,
		&author.StatusActivity,
		pq.Array(&author.Tags),
		&author.CVFile,
		&author.Version,
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
	err = stmt.QueryRowContext(ctx, args...).Scan(&author.UpdatedAt, &author.Version)
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
