package db

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"github.com/umtdemr/wb-backend/internal/token"
	"testing"
)

// TestRegisterUserTx tests registering user with transaction
func TestRegisterUserTx(t *testing.T) {
	args := RegisterUserTxParams{
		Email:        gofakeit.Email(),
		FullName:     gofakeit.Name(),
		PasswordHash: []byte(gofakeit.AppAuthor()),
		AuthProvider: "email",
	}

	result, err := testStore.RegisterUserTx(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, result.User.FullName, args.FullName)
	require.Equal(t, result.User.Email, args.Email)
	require.Equal(t, result.User.PasswordHash, args.PasswordHash)
	require.Equal(t, result.User.AuthProvider, args.AuthProvider)
	require.Equal(t, result.Token.UserID, int64(result.User.ID))
	require.Equal(t, result.Token.Scope, token.ScopeActivation)

	userWithQuery, err := testStore.GetUserByEmail(context.Background(), args.Email)
	require.NoError(t, err)
	require.Equal(t, userWithQuery.Email, result.User.Email)
}
