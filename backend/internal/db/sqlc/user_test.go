package db

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// createTestUser creates a random valid test user and tests it
func createTestUser(t *testing.T) *User {
	arg := CreateUserParams{
		Email:        gofakeit.Email(),
		FullName:     gofakeit.Name(),
		PasswordHash: []byte(gofakeit.AppAuthor()),
		AuthProvider: "email",
	}
	user, err := testStore.CreateUser(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, user)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.PasswordHash, user.PasswordHash)
	require.Equal(t, arg.AuthProvider, user.AuthProvider)
	require.Equal(t, user.Version, int32(1))

	require.WithinDuration(t, time.Now(), user.CreatedAt.Time, 1*time.Second)
	return &user
}

// TestCreateUser tests creating valid user
func TestCreateUser(t *testing.T) {
	createTestUser(t)
}

// TestUniqueEmail tests user creation with unique and not unique emails
func TestUniqueEmail(t *testing.T) {
	arg := CreateUserParams{
		Email:        gofakeit.Email(),
		FullName:     gofakeit.Name(),
		PasswordHash: []byte(gofakeit.AppAuthor()),
		AuthProvider: "email",
	}
	user1, err1 := testStore.CreateUser(context.Background(), arg)

	require.NoError(t, err1)
	require.NotEmpty(t, user1)

	// this user should not be created since the email is not unique
	user2, err2 := testStore.CreateUser(context.Background(), arg)
	require.Error(t, err2)
	require.Empty(t, user2)
}

// TestUpdateUser tests update user method with nullable params
func TestUpdateUser(t *testing.T) {
	user := createTestUser(t)

	updatedUser, err := testStore.UpdateUser(context.Background(), UpdateUserParams{
		FullName: pgtype.Text{String: "updated name", Valid: true},
		ID:       user.ID,
		Version:  user.Version,
	})

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	require.NotEqual(t, user.FullName, updatedUser.FullName)
	require.Equal(t, updatedUser.Version, user.Version+1)
	require.Equal(t, updatedUser.FullName, "updated name")
	require.Equal(t, updatedUser.Email, user.Email)
	require.Equal(t, updatedUser.IsVerified, user.IsVerified)
	require.Equal(t, updatedUser.PasswordHash, user.PasswordHash)
}

// TestGetForTokenParams tests inner join query for getting user from token data
func TestGetForTokenParams(t *testing.T) {
	token, user := createTokenForTesting(t)

	queryResponse, err := testStore.GetForToken(context.Background(), GetForTokenParams{
		Hash:   token.Hash,
		Scope:  token.Scope,
		Expiry: pgtype.Timestamptz{Time: token.Expiry.Time.Add(-3 * time.Second), Valid: true},
	})

	require.NoError(t, err)
	require.NotEmpty(t, queryResponse)
	require.NotEmpty(t, user)
	require.Equal(t, int64(user.ID), queryResponse.UserID)
	require.Equal(t, user.Email, queryResponse.Email)
	require.Equal(t, user.PasswordHash, queryResponse.PasswordHash)
	require.Equal(t, user.IsVerified, queryResponse.IsVerified)
	require.Equal(t, user.FullName, queryResponse.FullName)
	require.Equal(t, user.Version, queryResponse.Version)
}

// TestGetUserByEmail tests the query for getting the email with email
// It checks every field
func TestGetUserByEmail(t *testing.T) {
	user := createTestUser(t)

	queryUser, err := testStore.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	require.NotEmpty(t, queryUser)
	require.Equal(t, user.Email, queryUser.Email)
	require.Equal(t, user.ID, queryUser.ID)
	require.Equal(t, user.IsVerified, queryUser.IsVerified)
	require.Equal(t, user.FullName, queryUser.FullName)
	require.Equal(t, user.PasswordHash, queryUser.PasswordHash)
	require.Equal(t, user.CreatedAt, queryUser.CreatedAt)
	require.Equal(t, user.AuthProvider, queryUser.AuthProvider)
	require.Equal(t, user.AuthProviderID, queryUser.AuthProviderID)
	require.Equal(t, user.Version, queryUser.Version)
}
