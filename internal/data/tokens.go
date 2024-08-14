package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"github.com/deatil/go-encoding/base62"
	"time"
)

var (
	ErrDuplicateToken = errors.New("duplicate token")
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int       `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int, ttl time.Duration, scope string) (*Token, error) {

	// creating the token structure with basic data
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	// generating the token
	randomBytes := make([]byte, 64)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base62.StdEncoding.EncodeToString(randomBytes)

	// generating the hash to store in the DB
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

type TokenModel struct {
	db *sql.DB
}

func (m TokenModel) New(userID int, ttl time.Duration, scope string) (*Token, error) {

	// generating a new token
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	// saving it in the DB (and regenerate it if it's duplicated)
	err = m.Insert(token)
	if errors.Is(err, ErrDuplicateToken) {
		token, err = generateToken(userID, ttl, scope)
		if err != nil {
			return nil, err
		}

		err = m.Insert(token)
	}

	return token, err
}

func (m TokenModel) Insert(token *Token) error {

	// generating the query
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4);`

	// setting the arguments
	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

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
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "tokens_hash_key"`:
			return ErrDuplicateToken
		default:
			return err
		}
	}

	return nil
}

func (m TokenModel) DeleteAllForUser(scope string, userID int) error {

	// generating the query
	query := `
		DELETE FROM tokens
       	WHERE scope = $1 AND user_id = $2;`

	// setting the timeout context for the query execution
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if scope == "*" {
		// regenerating query
		query = `
			DELETE FROM tokens
       		WHERE user_id = ?;`

		// preparing the query
		stmt, err := m.db.PrepareContext(ctx, query)
		if err != nil {
			return err
		}
		defer stmt.Close()

		// executing the query
		_, err = m.db.ExecContext(ctx, query, userID)
		return err
	}

	// preparing the query
	stmt, err := m.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	// executing the query
	_, err = stmt.ExecContext(ctx, query, scope, userID)
	return err
}

func (m TokenModel) DeleteExpired() error {

	// generating the query
	query := `
		DELETE FROM tokens
		WHERE expiry < CURRENT_TIMESTAMP;`

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
	_, err = stmt.ExecContext(ctx)
	return err
}
