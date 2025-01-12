package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"github.com/umtdemr/wb-backend/internal/validator"
	"time"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// copyFromDbToken copies db token data into Token
func (t *Token) copyFromDbToken(dbToken *db.Token) {
	t.Scope = dbToken.Scope
	t.UserID = dbToken.UserID
	t.Expiry = dbToken.Expiry.Time
	t.Hash = dbToken.Hash
}

// ValidateTokenPlaintext validates the token's text
func ValidateTokenPlaintext(v *validator.Validator, tokenPlainString string) {
	v.Check(tokenPlainString != "", "token", "must be provided")
	v.Check(len(tokenPlainString) == 26, "token", "must be 26 bytes long")
}

type TokenModel interface {
	New(userId int64, ttl time.Duration, scope string) (*Token, error)
	Insert(token *Token) error
	DeleteAllForUser(scope string, userId int64) error
}

type DbTokenModel struct {
	store db.Store
}

// Ensure DbTokenModel implements TokenModel interface
var _ TokenModel = (*DbTokenModel)(nil)

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// New generates new token and inserts it
func (m *DbTokenModel) New(userId int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userId, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

// Insert adds token to the tokens table
func (m *DbTokenModel) Insert(token *Token) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.store.CreateToken(ctx, db.CreateTokenParams{
		Hash:   token.Hash,
		UserID: token.UserID,
		Expiry: pgtype.Timestamptz{Valid: true, Time: token.Expiry},
		Scope:  token.Scope,
	})
	return err
}

// DeleteAllForUser deletes all tokens for specific user
func (m *DbTokenModel) DeleteAllForUser(scope string, userId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.store.DeleteTokensForUser(ctx, db.DeleteTokensForUserParams{
		UserID: userId,
		Scope:  scope,
	})
}
