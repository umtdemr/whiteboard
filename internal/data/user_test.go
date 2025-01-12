package data

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	mockdb "github.com/umtdemr/wb-backend/internal/db/mock"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"github.com/umtdemr/wb-backend/internal/token"
	"reflect"
	"testing"
	"time"
)

// to test return error if no condition matches for the error
var unexpectedErr = errors.New("unexpected")

// TestPassword tests password set and matches methods
func TestPassword(t *testing.T) {
	plainPassword := "password123"

	passwordVar := password{}
	err := passwordVar.Set(plainPassword)
	require.NoError(t, err)

	require.Equal(t, plainPassword, *passwordVar.plaintext)

	ok, err := passwordVar.Matches(plainPassword)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestUser_IsAnonymous(t *testing.T) {
	require.Equal(t, AnonymousUser.IsAnonymous(), true)
}

// TestUser_CopyFromDbUser tests copying data from db user
func TestUser_CopyFromDbUser(t *testing.T) {
	dbUser := db.User{
		ID:           2,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		PasswordHash: []byte("passwordhash"),
		FullName:     "john doe",
		Email:        "john@doe.com",
		AuthProvider: "email",
		IsVerified:   pgtype.Bool{Valid: true, Bool: true},
		Version:      1,
	}

	user := &User{}
	user.CopyFromDbUser(&dbUser)

	require.Equal(t, user.ID, dbUser.ID)
	require.Equal(t, user.FullName, dbUser.FullName)
	require.Equal(t, user.Email, dbUser.Email)
	require.Equal(t, user.AuthProvider, dbUser.AuthProvider)
	require.Equal(t, user.IsVerified, dbUser.IsVerified.Bool)
	require.Equal(t, user.Version, int(dbUser.Version))
	require.Equal(t, user.CreatedAt, dbUser.CreatedAt.Time)
}

// TestUser_CopyFromDbUser tests copying data from db user
func TestUser_CopyFromDbJoinUser(t *testing.T) {
	dbUser := db.GetForTokenRow{
		UserID:       2,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		PasswordHash: []byte("passwordhash"),
		FullName:     "john doe",
		Email:        "john@doe.com",
		AuthProvider: "email",
		IsVerified:   pgtype.Bool{Valid: true, Bool: true},
		Version:      1,
	}

	user := &User{}
	user.CopyFromDbJoinUser(&dbUser)

	require.Equal(t, int64(user.ID), dbUser.UserID)
	require.Equal(t, user.FullName, dbUser.FullName)
	require.Equal(t, user.Email, dbUser.Email)
	require.Equal(t, user.AuthProvider, dbUser.AuthProvider)
	require.Equal(t, user.IsVerified, dbUser.IsVerified.Bool)
	require.Equal(t, user.Version, int(dbUser.Version))
	require.Equal(t, user.CreatedAt, dbUser.CreatedAt.Time)
}

// TestUserModel_Insert tests insert method of user model
func TestUserModel_Insert(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbUserModel{store}

	testCases := []struct {
		name          string
		input         *User
		buildStubs    func() // to build mock store
		checkResponse func(t *testing.T, user *User, err error)
	}{
		{
			name: "Successful insert",
			input: &User{
				FullName: "John Doe",
				Email:    "example@example.com",
				Password: password{hash: []byte("hashedpassword")},
			},
			buildStubs: func() {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:       1,
						FullName: "John Doe",
						Email:    "example@example.com",
					}, nil)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, user)
				require.Equal(t, "example@example.com", user.Email)
				require.Equal(t, "John Doe", user.FullName)
				require.Equal(t, 1, int(user.ID))
			},
		},
		{
			name: "Duplicate email",
			input: &User{
				FullName: "Jane Doe",
				Email:    "jane@example.com",
				Password: password{hash: []byte("hashedpassword")},
			},
			buildStubs: func() {
				mockErr := mockdb.MockPgError{ErrorCode: db.UniqueViolation}

				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, mockErr)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.Error(t, err)

				// it should raise duplicate email error
				require.EqualError(t, err, ErrDuplicateEmail.Error())
			},
		},
		{
			name: "unexpected error",
			input: &User{
				FullName: "Jane Doe",
				Email:    "jane@example.com",
				Password: password{hash: []byte("hashedpassword")},
			},
			buildStubs: func() {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, unexpectedErr)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.Error(t, err)
				require.EqualError(t, err, unexpectedErr.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs()

			user, err := model.Insert(tc.input)

			tc.checkResponse(t, user, err)
		})
	}
}

// TestUserModel_Update tests update method of user model
func TestUserModel_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbUserModel{store}

	testCases := []struct {
		name          string
		buildStubs    func()
		checkResponse func(t *testing.T, updatedUser *User, err error)
		updateParams  db.UpdateUserParams
	}{
		{
			name: "Succesful update",
			buildStubs: func() {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(db.UpdateUserParams{
						ID:       1,
						FullName: pgtype.Text{Valid: true, String: "john doe"},
						Email:    pgtype.Text{Valid: true, String: "john@doe.com"},
						Version:  1,
					})).
					Return(db.User{
						ID:       1,
						FullName: "john doe",
						Email:    "john@doe.com",
						Version:  2,
					}, nil)
			},
			checkResponse: func(t *testing.T, updatedUser *User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, updatedUser)
				require.Equal(t, int32(1), updatedUser.ID)
				require.Equal(t, "john doe", updatedUser.FullName)
				require.Equal(t, "john@doe.com", updatedUser.Email)
				require.Equal(t, 2, updatedUser.Version)
			},
			updateParams: db.UpdateUserParams{
				ID:       1,
				FullName: pgtype.Text{Valid: true, String: "john doe"},
				Email:    pgtype.Text{Valid: true, String: "john@doe.com"},
				Version:  1,
			},
		},
		{
			name: "Duplicate email",
			buildStubs: func() {
				uniqueViolationError := mockdb.MockPgError{ErrorCode: db.UniqueViolation}
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(db.UpdateUserParams{
						ID:       1,
						FullName: pgtype.Text{Valid: true, String: "john doe"},
						Email:    pgtype.Text{Valid: true, String: "john@doe.com"},
						Version:  1,
					})).
					Return(db.User{}, uniqueViolationError)
			},
			checkResponse: func(t *testing.T, updatedUser *User, err error) {
				require.Error(t, err)
				require.EqualError(t, err, ErrDuplicateEmail.Error())
				require.Nil(t, updatedUser)
			},
			updateParams: db.UpdateUserParams{
				ID:       1,
				FullName: pgtype.Text{Valid: true, String: "john doe"},
				Email:    pgtype.Text{Valid: true, String: "john@doe.com"},
				Version:  1,
			},
		},
		{
			name: "Edit conflict",
			buildStubs: func() {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, updatedUser *User, err error) {
				require.Error(t, err)
				require.EqualError(t, err, ErrEditConflict.Error())
				require.Nil(t, updatedUser)
			},
			updateParams: db.UpdateUserParams{
				ID:       1,
				FullName: pgtype.Text{Valid: true, String: "john doe"},
				Email:    pgtype.Text{Valid: true, String: "john@doe.com"},
				Version:  1,
			},
		},
		{
			name: "unexpected error",
			buildStubs: func() {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Return(db.User{}, unexpectedErr)
			},
			checkResponse: func(t *testing.T, updatedUser *User, err error) {
				require.Error(t, err)
				require.EqualError(t, err, unexpectedErr.Error())
				require.Nil(t, updatedUser)
			},
			updateParams: db.UpdateUserParams{
				ID:       1,
				FullName: pgtype.Text{Valid: true, String: "john doe"},
				Email:    pgtype.Text{Valid: true, String: "john@doe.com"},
				Version:  1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs()

			user, err := model.Update(tc.updateParams)

			tc.checkResponse(t, user, err)
		})
	}
}

// TestUserModel_GetByEmail tests retrieving user with email via user model
func TestUserModel_GetByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbUserModel{store}
	testCases := []struct {
		name          string
		buildStubs    func()
		checkResponse func(t *testing.T, user *User, err error)
	}{
		{
			name: "Successful retrieve",
			buildStubs: func() {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq("johndoe@gmail.com")).
					Return(
						db.User{
							ID:       3,
							FullName: "john doe",
							Email:    "johndoe@gmail.com",
							Version:  4,
						},
						nil,
					)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, user)
				require.Equal(t, 3, int(user.ID))
				require.Equal(t, "john doe", user.FullName)
				require.Equal(t, "johndoe@gmail.com", user.Email)
				require.Equal(t, 4, user.Version)
			},
		},
		{
			name: "Not found",
			buildStubs: func() {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq("johndoe@gmail.com")).
					Return(
						db.User{},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.Error(t, err)
				require.Empty(t, user)
				require.EqualError(t, err, ErrRecordNotFound.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs()

			user, err := model.GetByEmail("johndoe@gmail.com")

			tc.checkResponse(t, user, err)
		})
	}
}

// getForTokenParamsMatcher is a custom matcher for checking params for TestUserModel_GetForToken
type getForTokenParamsMatcher struct {
	expected db.GetForTokenParams
}

func (m getForTokenParamsMatcher) Matches(x interface{}) bool {
	actual, ok := x.(db.GetForTokenParams)
	if !ok {
		return false
	}

	if m.expected.Scope != actual.Scope {
		return false
	}
	if !reflect.DeepEqual(m.expected.Hash, actual.Hash) {
		return false
	}

	if m.expected.Expiry.Valid != actual.Expiry.Valid {
		return false
	}
	if m.expected.Expiry.Valid && actual.Expiry.Valid {
		diff := m.expected.Expiry.Time.Sub(actual.Expiry.Time)
		if diff < -time.Second || diff > time.Second {
			return false
		}
	}

	return true
}

func (m getForTokenParamsMatcher) String() string {
	return fmt.Sprintf("is equal to %v (ignoring small time differences)", m.expected)
}

// EqGetForTokenParams checks if the db.GetForTokenParams values are expected
func EqGetForTokenParams(params db.GetForTokenParams) gomock.Matcher {
	return getForTokenParamsMatcher{expected: params}
}

// TestUserModel_GetForToken tests retrieving user by token scope and token
func TestUserModel_GetForToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbUserModel{store}

	type params struct {
		scope string
		token string
	}

	testCases := []struct {
		name          string
		buildStubs    func()
		checkResponse func(t *testing.T, user *User, err error)
		getParams     params
	}{
		{
			name: "success retrieve",
			buildStubs: func() {
				tokenHash := sha256.Sum256([]byte("tokenfortesting"))
				Expiry := pgtype.Timestamptz{Time: time.Now(), Valid: true}

				store.EXPECT().
					GetForToken(gomock.Any(), EqGetForTokenParams(
						db.GetForTokenParams{
							Scope:  "test",
							Hash:   tokenHash[:],
							Expiry: Expiry,
						},
					)).
					Return(
						db.GetForTokenRow{
							UserID:   3,
							FullName: "john doe",
							Email:    "johndoe@gmail.com",
							Version:  4,
						},
						nil,
					)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, user)
			},
			getParams: params{token: "tokenfortesting", scope: "test"},
		},
		{
			name: "success retrieve",
			buildStubs: func() {
				tokenHash := sha256.Sum256([]byte("tokenfortesting"))
				Expiry := pgtype.Timestamptz{Time: time.Now(), Valid: true}

				store.EXPECT().
					GetForToken(gomock.Any(), EqGetForTokenParams(
						db.GetForTokenParams{
							Scope:  "test",
							Hash:   tokenHash[:],
							Expiry: Expiry,
						},
					)).
					Return(
						db.GetForTokenRow{
							UserID:   3,
							FullName: "john doe",
							Email:    "johndoe@gmail.com",
							Version:  4,
						},
						nil,
					)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, user)
			},
			getParams: params{token: "tokenfortesting", scope: "test"},
		},
		{
			name: "not found",
			buildStubs: func() {
				store.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(
						db.GetForTokenRow{},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.Error(t, err)
				require.Empty(t, user)
				require.EqualError(t, err, ErrRecordNotFound.Error())
			},
			getParams: params{token: "tokenfortesting", scope: "test"},
		},
		{
			name: "unexpected error",
			buildStubs: func() {
				store.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(
						db.GetForTokenRow{},
						unexpectedErr,
					)
			},
			checkResponse: func(t *testing.T, user *User, err error) {
				require.Error(t, err)
				require.Empty(t, user)
				require.EqualError(t, err, unexpectedErr.Error())
			},
			getParams: params{token: "tokenfortesting", scope: "test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs()

			user, err := model.GetForToken(tc.getParams.scope, tc.getParams.token)

			tc.checkResponse(t, user, err)
		})
	}
}

// getForTokenParamsMatcher is a custom matcher for checking params for TestUserModel_GetForToken
type registerUserTxParamsMatcher struct {
	expected db.RegisterUserTxParams
}

func (m registerUserTxParamsMatcher) Matches(x interface{}) bool {
	actual, ok := x.(db.RegisterUserTxParams)
	if !ok {
		return false
	}

	if m.expected.FullName != actual.FullName {
		return false
	}
	if m.expected.Email != actual.Email {
		return false
	}
	if !reflect.DeepEqual(m.expected.PasswordHash, actual.PasswordHash) {
		return false
	}

	if m.expected.AuthProvider != actual.AuthProvider {
		return false
	}
	return true
}

func (m registerUserTxParamsMatcher) String() string {
	return fmt.Sprintf("is equal to %v (ignoring small time differences)", m.expected)
}

// EqGetForTokenParams checks if the db.GetForTokenParams values are expected
func EqRegisterUserTxParams(params db.RegisterUserTxParams) gomock.Matcher {
	return registerUserTxParamsMatcher{expected: params}
}

// TestUserModel_Register tests register transaction for user model
func TestUserModel_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbUserModel{store}

	type params struct {
		FullName     string
		Email        string
		PasswordHash []byte
		AuthProvider string
	}

	testCases := []struct {
		name           string
		buildStub      func()
		checkResponse  func(t *testing.T, result *RegisterUserResult, err error)
		registerParams params
	}{
		{
			name: "successful register",
			buildStub: func() {
				store.EXPECT().
					RegisterUserTx(gomock.Any(), EqRegisterUserTxParams(db.RegisterUserTxParams{
						Email:        "johndoe@gmail.com",
						FullName:     "john doe",
						PasswordHash: []byte("secretpassowrd"),
						AuthProvider: "email",
					})).
					Return(db.RegisterUserTxResult{
						User: db.User{
							ID:           10,
							Email:        "johndoe@gmail.com",
							FullName:     "john doe",
							AuthProvider: "email",
						},
						Token: db.Token{
							Hash:   []byte("secrettoken")[:],
							Expiry: pgtype.Timestamptz{Valid: true, Time: time.Now().Add(24 * time.Hour)},
							UserID: 10,
							Scope:  token.ScopeActivation,
						},
						TokenPlaintext: "secrettoken",
					}, nil)
			},
			checkResponse: func(t *testing.T, result *RegisterUserResult, err error) {
				require.NoError(t, err)
			},
			registerParams: params{
				FullName:     "john doe",
				Email:        "johndoe@gmail.com",
				AuthProvider: "email",
				PasswordHash: []byte("secretpassowrd"),
			},
		},
		{
			name: "Duplicate email",
			buildStub: func() {
				mockErr := mockdb.MockPgError{ErrorCode: db.UniqueViolation}
				store.EXPECT().RegisterUserTx(gomock.Any(), gomock.Any()).
					Return(db.RegisterUserTxResult{}, mockErr)
			},
			checkResponse: func(t *testing.T, result *RegisterUserResult, err error) {
				require.Error(t, err)
				require.EqualError(t, err, ErrDuplicateEmail.Error())
			},
			registerParams: params{
				FullName:     "john doe",
				Email:        "johndoe@gmail.com",
				AuthProvider: "email",
				PasswordHash: []byte("secretpassowrd"),
			},
		},
		{
			name: "Unexpected error",
			buildStub: func() {
				store.EXPECT().RegisterUserTx(gomock.Any(), gomock.Any()).
					Return(db.RegisterUserTxResult{}, unexpectedErr)
			},
			checkResponse: func(t *testing.T, result *RegisterUserResult, err error) {
				require.Error(t, err)
				require.EqualError(t, err, unexpectedErr.Error())
			},
			registerParams: params{
				FullName:     "john doe",
				Email:        "johndoe@gmail.com",
				AuthProvider: "email",
				PasswordHash: []byte("secretpassowrd"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			user := &User{
				FullName:     tc.registerParams.FullName,
				Email:        tc.registerParams.Email,
				Password:     password{hash: tc.registerParams.PasswordHash},
				AuthProvider: tc.registerParams.AuthProvider,
			}
			result, err := model.Register(user)

			tc.checkResponse(t, result, err)
		})
	}
}

// TestDbUserModel_ActivateUser tests activating user via UserModel.ActivateUser
func TestDbUserModel_ActivateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbUserModel{store}

	var unexpectedErr = errors.New("unexpected")

	testCases := []struct {
		name          string
		buildStub     func()
		checkResponse func(t *testing.T, result *ActivateUserResult, err error)
		token         string
	}{
		{
			name: "error token fetch - not found",
			buildStub: func() {
				store.EXPECT().
					ActivateUserTx(gomock.Any(), gomock.Eq("")).
					Return(
						db.ActivateUserTxResult{
							ErrTokenFetch: pgx.ErrNoRows,
						},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, result *ActivateUserResult, err error) {
				require.Error(t, err)
				require.EqualError(t, ErrRecordNotFound, result.ErrTokenFetch.Error())
			},
			token: "",
		},
		{
			name: "error token fetch - unexpected error",
			buildStub: func() {
				store.EXPECT().
					ActivateUserTx(gomock.Any(), gomock.Eq("")).
					Return(
						db.ActivateUserTxResult{
							ErrTokenFetch: unexpectedErr,
						},
						unexpectedErr,
					)
			},
			checkResponse: func(t *testing.T, result *ActivateUserResult, err error) {
				require.Error(t, err)
				require.EqualError(t, unexpectedErr, result.ErrTokenFetch.Error())
			},
			token: "",
		},
		{
			name: "error updating user - conflict error",
			buildStub: func() {
				store.EXPECT().
					ActivateUserTx(gomock.Any(), gomock.Eq("")).
					Return(
						db.ActivateUserTxResult{
							ErrUpdateUser: pgx.ErrNoRows,
						},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, result *ActivateUserResult, err error) {
				require.Error(t, err)
				require.Nil(t, result.ErrTokenFetch)
				require.EqualError(t, ErrEditConflict, result.ErrUpdateUser.Error())
			},
			token: "",
		},
		{
			name: "error updating user - unexpected error",
			buildStub: func() {
				store.EXPECT().
					ActivateUserTx(gomock.Any(), gomock.Eq("")).
					Return(
						db.ActivateUserTxResult{
							ErrUpdateUser: unexpectedErr,
						},
						unexpectedErr,
					)
			},
			checkResponse: func(t *testing.T, result *ActivateUserResult, err error) {
				require.Error(t, err)
				require.Nil(t, result.ErrTokenFetch)
				require.EqualError(t, unexpectedErr, result.ErrUpdateUser.Error())
			},
			token: "",
		},
		{
			name: "successful activate",
			buildStub: func() {
				store.EXPECT().
					ActivateUserTx(gomock.Any(), gomock.Eq("VALID-26-bytes-token")).
					Return(
						db.ActivateUserTxResult{
							User: db.User{
								ID:         1,
								FullName:   "full_name",
								Email:      "test@test.com",
								IsVerified: pgtype.Bool{Valid: true, Bool: true},
							},
						},
						nil,
					)
			},
			checkResponse: func(t *testing.T, result *ActivateUserResult, err error) {
				require.NoError(t, err)

				require.NotEmpty(t, result.User)
				require.Equal(t, int32(1), result.User.ID)
				require.Equal(t, "full_name", result.User.FullName)
				require.Equal(t, "test@test.com", result.User.Email)
				require.True(t, result.User.IsVerified)

			},
			token: "VALID-26-bytes-token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			result, err := model.ActivateUser(tc.token)

			tc.checkResponse(t, result, err)
		})
	}
}
