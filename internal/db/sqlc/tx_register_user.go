package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/umtdemr/wb-backend/internal/token"
	"time"
)

type RegisterUserTxParams struct {
	FullName     string
	Email        string
	PasswordHash []byte
	AuthProvider string
}

type RegisterUserTxResult struct {
	User           User
	Token          Token
	TokenPlaintext string
}

func (s *SQLStore) RegisterUserTx(ctx context.Context, params RegisterUserTxParams) (RegisterUserTxResult, error) {
	var result RegisterUserTxResult

	err := s.execTx(ctx, func(queries *Queries) error {
		var err error
		result.User, err = queries.CreateUser(
			ctx,
			CreateUserParams{
				FullName:     params.FullName,
				Email:        params.Email,
				PasswordHash: params.PasswordHash,
				AuthProvider: "email",
				IsVerified:   pgtype.Bool{Valid: true, Bool: true},
			},
		)

		if err != nil {
			return err
		}

		_, err = queries.AddForUserWithCode(ctx, AddForUserWithCodeParams{UserID: int64(result.User.ID), Codes: []string{"healtcheck:read"}})

		if err != nil {
			return err
		}

		var tokenHash []byte
		result.TokenPlaintext, tokenHash, err = token.GenerateToken()

		if err != nil {
			return err
		}

		result.Token, err = queries.CreateToken(ctx, CreateTokenParams{
			UserID: int64(result.User.ID),
			Hash:   tokenHash,
			Scope:  token.ScopeActivation,
			Expiry: pgtype.Timestamptz{Valid: true, Time: time.Now().Add(24 * time.Hour)},
		})

		return err
	})

	return result, err
}
