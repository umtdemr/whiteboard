package data

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"github.com/umtdemr/wb-backend/internal/validator"
	"golang.org/x/crypto/bcrypt"
	"time"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

// AnonymousUser represents inactivated user with no Id, name, email or password
var AnonymousUser = &User{}

type User struct {
	ID           int32     `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	Password     password  `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	AuthProvider string    `json:"auth_provider"`
	Version      int       `json:"version"`
	IsVerified   bool      `json:"is_verified"`
}

// IsAnonymous checks if the user is anonymous
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

func (u *User) CopyFromDbUser(dbUser *db.User) {
	u.ID = dbUser.ID
	u.Email = dbUser.Email
	u.FullName = dbUser.FullName
	u.Password.hash = dbUser.PasswordHash
	u.AuthProvider = dbUser.AuthProvider
	u.IsVerified = dbUser.IsVerified.Bool
	u.CreatedAt = dbUser.CreatedAt.Time
	u.Version = int(dbUser.Version)
}

func (u *User) CopyFromDbJoinUser(dbUser *db.GetForTokenRow) {
	u.ID = int32(dbUser.UserID)
	u.Email = dbUser.Email
	u.FullName = dbUser.FullName
	u.Password.hash = dbUser.PasswordHash
	u.AuthProvider = dbUser.AuthProvider
	u.IsVerified = dbUser.IsVerified.Bool
	u.CreatedAt = dbUser.CreatedAt.Time
	u.Version = int(dbUser.Version)
}

type password struct {
	plaintext *string
	hash      []byte
}

// Set hashes plain password and saves both of them
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)

	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

// Matches checks if the given plain password is correct
func (p *password) Matches(plaintextString string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextString))
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
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be valid email")
}

func ValidatePasswordPlainText(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8 && len(password) <= 72, "password", "must be between 8 and 72")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.FullName != "", "full_name", "must be provided")
	v.Check(len(user.FullName) >= 3 && len(user.FullName) <= 25, "full_name", "must between 3 and 25")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlainText(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UserModel interface {
	Insert(user *User) (*User, error)
	Update(params db.UpdateUserParams) (*User, error)
	GetByEmail(email string) (*User, error)
	GetForToken(scope string, token string) (*User, error)
	Register(user *User) (*RegisterUserResult, error)
	ActivateUser(token string) (*ActivateUserResult, error)
}

type DbUserModel struct {
	store db.Store
}

// Ensure DbUserModel implements UserModel interface
var _ UserModel = (*DbUserModel)(nil)

func (m *DbUserModel) Insert(user *User) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	createdUser, err := m.store.CreateUser(
		ctx,
		db.CreateUserParams{
			FullName:     user.FullName,
			Email:        user.Email,
			PasswordHash: user.Password.hash,
			AuthProvider: "email",
			IsVerified:   pgtype.Bool{Valid: true, Bool: user.IsVerified},
		},
	)

	if err != nil {
		if db.IsErrUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	returningUser := &User{}
	returningUser.CopyFromDbUser(&createdUser)

	return returningUser, nil
}

func (m *DbUserModel) Update(params db.UpdateUserParams) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user := &User{}

	dbUser, err := m.store.UpdateUser(ctx, params)

	if err != nil {
		switch {
		case db.IsErrUniqueViolation(err):
			return nil, ErrDuplicateEmail
		case db.IsErrNoRows(err):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}

	user.CopyFromDbUser(&dbUser)

	return user, nil
}

// GetByEmail finds and returns user by email
func (m *DbUserModel) GetByEmail(email string) (*User, error) {
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fetchedUser, err := m.store.GetUserByEmail(ctx, email)

	if err != nil {
		if db.IsErrNoRows(err) {
			return nil, ErrRecordNotFound
		}
	}

	// copy the values from db model to our model
	user.CopyFromDbUser(&fetchedUser)

	return &user, nil
}

// GetForToken finds user with token data
func (m *DbUserModel) GetForToken(scope string, token string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(token))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user := &User{}

	dbUser, err := m.store.GetForToken(ctx, db.GetForTokenParams{
		Scope:  scope,
		Hash:   tokenHash[:],
		Expiry: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	user.CopyFromDbJoinUser(&dbUser)
	return user, nil
}

type RegisterUserResult struct {
	User           *User
	Token          *Token
	TokenPlaintext string
}

// Register is a transaction handler for registering
func (m *DbUserModel) Register(user *User) (*RegisterUserResult, error) {
	dataResult := &RegisterUserResult{}

	dbResult, err := m.store.RegisterUserTx(context.Background(), db.RegisterUserTxParams{
		Email:        user.Email,
		FullName:     user.FullName,
		PasswordHash: user.Password.hash,
		AuthProvider: user.AuthProvider,
	})

	if err != nil && db.IsErrUniqueViolation(err) {
		return dataResult, ErrDuplicateEmail
	}

	registeredUser := &User{}
	registeredUser.CopyFromDbUser(&dbResult.User)

	token := &Token{}
	token.copyFromDbToken(&dbResult.Token)

	dataResult.User = registeredUser
	dataResult.Token = token
	dataResult.TokenPlaintext = dbResult.TokenPlaintext

	return dataResult, err
}

type ActivateUserResult struct {
	User                   *User
	ErrTokenFetch          error
	ErrUpdateUser          error
	ErrDeleteTokensForUser error
}

// ActivateUser is a transaction handler for activating a user with token
func (m *DbUserModel) ActivateUser(token string) (*ActivateUserResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbResult, txErr := m.store.ActivateUserTx(ctx, token)
	activatedUser := &User{}
	activatedUser.CopyFromDbUser(&dbResult.User)

	// handle transaction errors accurately

	// handle error for db.store.GetForToken query
	var errTokenFetch error

	if dbResult.ErrTokenFetch != nil {
		switch {
		case errors.Is(dbResult.ErrTokenFetch, pgx.ErrNoRows):
			errTokenFetch = ErrRecordNotFound
		default:
			errTokenFetch = dbResult.ErrTokenFetch
		}
	}

	// handle error for updating user
	var errUpdateUser error
	if dbResult.ErrUpdateUser != nil {
		switch {
		case db.IsErrNoRows(dbResult.ErrUpdateUser):
			errUpdateUser = ErrEditConflict
		default:
			errUpdateUser = dbResult.ErrUpdateUser
		}
	}

	var activateUserResult = ActivateUserResult{
		User:                   activatedUser,
		ErrTokenFetch:          errTokenFetch,
		ErrUpdateUser:          errUpdateUser,
		ErrDeleteTokensForUser: dbResult.ErrDeleteTokensForUser,
	}

	return &activateUserResult, txErr
}
