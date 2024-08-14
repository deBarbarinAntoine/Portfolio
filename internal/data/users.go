package data

import (
	"Portfolio/internal/validator"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	AnonymousUser     = &User{}
)

type User struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Avatar    string    `json:"avatar,omitempty"`
	Status    string    `json:"status"`
	Version   int       `json:"-"`
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plainTextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainTextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plainTextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plainTextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plainTextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must not be more than 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "username", "must be provided")
	v.Check(len(user.Name) <= 500, "username", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UserModel struct {
	db *sql.DB
}

func (m UserModel) Insert(user *User) error {

	// creating the query
	query := `
		INSERT INTO users (name, email, password_hash, avatar, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version;`

	// setting the arguments
	args := []any{user.Name, user.Email, user.Password.hash, user.Avatar, user.Status}

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
	err = stmt.QueryRowContext(ctx, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Update(user *User) error {

	// creating the query
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, avatar = $4, status = $5, version = version + 1
		WHERE id = $6 AND version = $7
		RETURNING version;`

	// setting the arguments
	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Avatar,
		user.Status,
		user.ID,
		user.Version,
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
	err = stmt.QueryRowContext(ctx, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Delete(user *User) error {

	// creating the query
	query := `
		DELETE FROM users
		WHERE id = $1 AND version = $2;`

	// setting the arguments
	args := []any{
		user.ID,
		user.Version,
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
	_, err = stmt.ExecContext(ctx, args...)

	return err
}

func (m UserModel) DeleteExpired() error {

	// generating the query
	query := `
		DELETE FROM users u
		USING tokens t
		WHERE u.id = t.user_id AND u.status = $1 AND (t.expiry IS NULL OR t.expiry < CURRENT_TIMESTAMP);`

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
	_, err = stmt.ExecContext(ctx, UserToActivate)
	if err != nil {
		return fmt.Errorf("failed to delete expired users: %w", err)
	}

	return nil
}

func (m UserModel) Exists(id int) (bool, error) {

	// creating the query
	query := `
		SELECT EXISTS (
		SELECT 1 FROM users WHERE id = $1);`

	// setting the timeout for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	// executing the query
	var exists bool
	err = stmt.QueryRowContext(ctx, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (m UserModel) GetByID(id int) (*User, error) {

	// creating the query
	query := `
		SELECT id, created_at, name, email, password_hash, avatar, status, version
		FROM users
		WHERE id = $1;`

	// setting the user variable
	var user User

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// executing the query
	err = stmt.QueryRowContext(ctx, id).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Avatar,
		&user.Status,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) GetByEmail(email string) (*User, error) {

	// creating the query
	query := `
		SELECT id, created_at, name, email, password_hash, avatar, status, version
		FROM users
		WHERE email = $1;`

	// setting the user variable
	var user User

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// executing the query
	err = stmt.QueryRowContext(ctx, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Avatar,
		&user.Status,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {

	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	// creating the query
	query := `
		SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.avatar, users.status, users.version
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3;`

	// setting the arguments
	args := []any{tokenHash[:], tokenScope, time.Now()}

	// setting the user variable
	var user User

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// executing the query
	err = stmt.QueryRowContext(ctx, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Avatar,
		&user.Status,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
