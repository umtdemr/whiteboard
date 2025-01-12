package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines all the methods to execute db queries
type Store interface {
	Querier
	RegisterUserTx(ctx context.Context, params RegisterUserTxParams) (RegisterUserTxResult, error)
	CreateBoardTx(ctx context.Context, params CreateBoardTxParams) (CreateBoardTxResult, error)
	ActivateUserTx(ctx context.Context, plainToken string) (ActivateUserTxResult, error)
}

type SQLStore struct {
	DB *pgxpool.Pool
	*Queries
}

func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		DB:      connPool,
		Queries: New(connPool),
	}
}

// execTx wraps queries with transaction and executes them
func (s *SQLStore) execTx(ctx context.Context, dbQueryFn func(*Queries) error) error {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})

	if err != nil {
		return err
	}

	queriesTx := New(tx)
	err = dbQueryFn(queriesTx)

	if err != nil {
		rbErr := tx.Rollback(ctx)
		if rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit(ctx)
}
