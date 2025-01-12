package db

import (
	"context"
	"crypto/sha256"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/umtdemr/wb-backend/internal/token"
	"time"
)

type ActivateUserTxResult struct {
	User                   User
	ErrTokenFetch          error
	ErrUpdateUser          error
	ErrDeleteTokensForUser error
}

func (s *SQLStore) ActivateUserTx(ctx context.Context, plainToken string) (ActivateUserTxResult, error) {
	var result ActivateUserTxResult

	err := s.execTx(ctx, func(queries *Queries) error {
		tokenHash := sha256.Sum256([]byte(plainToken))
		tokenRow, err := queries.GetForToken(ctx, GetForTokenParams{
			Scope:  token.ScopeActivation,
			Hash:   tokenHash[:],
			Expiry: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})

		if err != nil {
			result.ErrTokenFetch = err
			return err
		}

		result.User = User{
			ID:             int32(tokenRow.UserID),
			Version:        tokenRow.Version,
			FullName:       tokenRow.FullName,
			CreatedAt:      tokenRow.CreatedAt,
			Email:          tokenRow.Email,
			PasswordHash:   tokenRow.PasswordHash,
			IsVerified:     tokenRow.IsVerified,
			AuthProvider:   tokenRow.AuthProvider,
			AuthProviderID: tokenRow.AuthProviderID,
		}

		_, err = queries.UpdateUser(ctx, UpdateUserParams{
			ID:         result.User.ID,
			Version:    tokenRow.Version,
			IsVerified: pgtype.Bool{Bool: true, Valid: true},
		})

		if err != nil {
			result.ErrUpdateUser = err
			return err
		}

		err = queries.DeleteTokensForUser(ctx, DeleteTokensForUserParams{
			UserID: tokenRow.UserID,
			Scope:  token.ScopeActivation,
		})

		if err != nil {
			result.ErrDeleteTokensForUser = err
		}

		return err
	})

	return result, err
}
