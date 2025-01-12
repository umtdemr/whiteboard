package db

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func createTokenForTesting(t *testing.T) (*Token, *User) {
	tokenExpiry := time.Now().Add(2 * time.Hour)
	user := createTestUser(t)
	var userId int64 = int64(user.ID)
	hash := []byte(gofakeit.Name())
	scope := "testing"
	token, err := testStore.CreateToken(context.Background(), CreateTokenParams{
		Hash:   hash,
		UserID: userId,
		Expiry: pgtype.Timestamptz{Time: tokenExpiry, Valid: true},
		Scope:  scope,
	})

	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, token.Hash, hash)
	require.WithinDuration(t, time.Now().Add(2*time.Hour), token.Expiry.Time, 3*time.Second)
	require.Equal(t, token.Scope, scope)

	return &token, user

}

func TestCreateToken(t *testing.T) {
	createTokenForTesting(t)
}
