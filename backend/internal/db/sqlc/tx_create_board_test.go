package db

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

// TestCreateBoardTx tests creating board and its page using Store.CreateBoardTx
func TestCreateBoardTx(t *testing.T) {
	user := createTestUser(t)

	testCases := []struct {
		name    string
		handler func(t *testing.T)
	}{
		{
			name: "With default page name",
			handler: func(t *testing.T) {
				args := CreateBoardTxParams{
					Name:    gofakeit.LetterN(10),
					OwnerId: int64(user.ID),
					SlugId:  gofakeit.LetterN(10),
				}
				result, err := testStore.CreateBoardTx(
					context.Background(),
					args,
				)

				require.NoError(t, err)
				require.NotEmpty(t, result)

				require.Equal(t, result.Board.Name, args.Name)
				require.Equal(t, result.Board.OwnerID, args.OwnerId)
				require.Equal(t, result.Board.SlugID, args.SlugId)
				require.Equal(t, result.Board.IsDeleted, false)
				require.WithinDuration(t, result.Board.CreatedAt.Time, time.Now(), time.Second)

				require.Equal(t, result.Page.BoardID, int64(result.Board.ID))
				require.Equal(t, result.Page.Name, DefaultBoardPageName)
				require.WithinDuration(t, result.Page.CreatedAt.Time, time.Now(), time.Second)
			},
		},
		{
			name: "With custom page name",
			handler: func(t *testing.T) {
				customPageName := gofakeit.LetterN(10)

				args := CreateBoardTxParams{
					Name:     gofakeit.LetterN(10),
					OwnerId:  int64(user.ID),
					SlugId:   gofakeit.LetterN(10),
					PageName: &customPageName,
				}
				result, err := testStore.CreateBoardTx(
					context.Background(),
					args,
				)

				require.NoError(t, err)
				require.Equal(t, result.Page.Name, customPageName)
			},
		},
		{
			name: "Duplicate slug ID",
			handler: func(t *testing.T) {

				args := CreateBoardTxParams{
					Name:    gofakeit.LetterN(10),
					OwnerId: int64(user.ID),
					SlugId:  gofakeit.LetterN(10),
				}
				_, err := testStore.CreateBoardTx(
					context.Background(),
					args,
				)

				require.NoError(t, err)

				_, err = testStore.CreateBoardTx(
					context.Background(),
					args,
				)

				require.Error(t, err)
				require.True(t, IsErrUniqueViolation(err))
			},
		},
		{
			name: "Invalid param",
			handler: func(t *testing.T) {

				args := CreateBoardTxParams{
					Name:    gofakeit.Name(),
					OwnerId: -23,
					SlugId:  gofakeit.LetterN(10),
				}
				_, err := testStore.CreateBoardTx(
					context.Background(),
					args,
				)

				require.Error(t, err)
				require.True(t, IsErrForeignKeyViolation(err))
			},
		},
		{
			name: "Transaction Integrity",
			handler: func(t *testing.T) {
				args := CreateBoardTxParams{
					Name:    gofakeit.LetterN(10),
					OwnerId: int64(user.ID),
					SlugId:  gofakeit.LetterN(10),
				}
				pageName := strings.Repeat("a", 1000)
				args.PageName = &pageName

				_, err := testStore.CreateBoardTx(context.Background(), args)
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.handler(t)
		})
	}
}
