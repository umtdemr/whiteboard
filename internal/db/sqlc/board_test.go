package db

import (
	"context"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func createTestingBoard(t *testing.T, user *User) *Board {
	args := CreateBoardParams{
		OwnerID: int64(user.ID),
		Name:    gofakeit.Name(),
		SlugID:  gofakeit.LetterN(10),
	}

	board, err := testStore.CreateBoard(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.NotEmpty(t, board)
	require.Equal(t, args.OwnerID, board.OwnerID)
	require.Equal(t, args.Name, board.Name)
	require.Equal(t, args.SlugID, board.SlugID)
	require.Equal(t, args.SlugID, board.SlugID)
	//require.WithinDuration(t, board.CreatedAt.Time, time.Now(), 1*time.Second)
	require.False(t, board.IsDeleted)

	return &board
}

// TestCreateBoard tests happy and unhappy case for creating boards
func TestCreateBoard(t *testing.T) {
	testCases := []struct {
		name    string
		handler func(t *testing.T)
	}{
		{
			name: "Successful insert",
			handler: func(t *testing.T) {
				user := createTestUser(t)
				board := createTestingBoard(t, user)
				require.NotNil(t, board)
			},
		},
		{
			name: "Error on duplicate slug id",
			handler: func(t *testing.T) {
				user := createTestUser(t)
				board := createTestingBoard(t, user)
				require.NotNil(t, board)
				board2, err := testStore.CreateBoard(
					context.Background(),
					CreateBoardParams{
						OwnerID: int64(user.ID),
						Name:    gofakeit.Name(),
						SlugID:  board.SlugID,
					},
				)

				require.Error(t, err)
				require.True(t, IsErrUniqueViolation(err))
				require.Empty(t, board2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.handler(t)
		})
	}
}

func createTestingBoardPage(t *testing.T, board *Board) *BoardPage {
	args := CreateBoardPageParams{
		BoardID: int64(board.ID),
		Name:    gofakeit.LetterN(20),
	}

	boardPage, err := testStore.CreateBoardPage(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, boardPage)
	require.Equal(t, boardPage.BoardID, int64(board.ID))
	require.WithinDuration(t, boardPage.CreatedAt.Time, time.Now(), time.Second)
	require.False(t, boardPage.IsDeleted)

	return &boardPage
}

// TestCreateBoardPage tests creating board page
func TestCreateBoardPage(t *testing.T) {
	user := createTestUser(t)
	board := createTestingBoard(t, user)
	createTestingBoardPage(t, board)
}

// TestAddToBoardUsers tests creating a board_user row
func TestAddToBoardUsers(t *testing.T) {
	testCases := []struct {
		name    string
		handler func(t *testing.T)
	}{
		{
			name: "Successful insert",
			handler: func(t *testing.T) {
				user := createTestUser(t)
				board := createTestingBoard(t, user)
				args := AddToBoardUsersParams{
					UserID:  int64(user.ID),
					BoardID: int64(board.ID),
					Role:    BoardRoleEditor,
				}

				boardUserRow, err := testStore.AddToBoardUsers(context.Background(), args)
				require.NoError(t, err)
				require.NotEmpty(t, boardUserRow)
				require.Equal(t, args.UserID, boardUserRow.UserID)
				require.Equal(t, args.BoardID, boardUserRow.BoardID)
				require.Equal(t, args.Role, boardUserRow.Role)
			},
		},
		{
			name: "Invalid role",
			handler: func(t *testing.T) {
				user := createTestUser(t)
				board := createTestingBoard(t, user)
				args := AddToBoardUsersParams{
					UserID:  int64(user.ID),
					BoardID: int64(board.ID),
					Role:    "invalid",
				}

				boardUserRow, err := testStore.AddToBoardUsers(context.Background(), args)
				require.Error(t, err)
				require.Empty(t, boardUserRow)
			},
		},
		{
			name: "Duplicate board_user data",
			handler: func(t *testing.T) {
				user := createTestUser(t)
				board := createTestingBoard(t, user)
				args := AddToBoardUsersParams{
					UserID:  int64(user.ID),
					BoardID: int64(board.ID),
					Role:    BoardRoleEditor,
				}

				boardUserRow, err := testStore.AddToBoardUsers(context.Background(), args)
				require.NoError(t, err)
				require.NotEmpty(t, boardUserRow)

				boardUserRow2, err := testStore.AddToBoardUsers(context.Background(), args)
				require.Error(t, err)
				require.Empty(t, boardUserRow2)
				require.True(t, IsErrUniqueViolation(err))
			},
		},
		{
			name: "Invalid foreign key",
			handler: func(t *testing.T) {
				user := createTestUser(t)
				board := createTestingBoard(t, user)
				args := AddToBoardUsersParams{
					UserID:  -1,
					BoardID: int64(board.ID),
					Role:    BoardRoleEditor,
				}

				boardUserRow, err := testStore.AddToBoardUsers(context.Background(), args)
				require.Error(t, err)
				require.Empty(t, boardUserRow)
				require.True(t, IsErrForeignKeyViolation(err))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.handler(t)
		})
	}
}

// TestGetAllBoardsForUer tests retrieving all boards for a user
func TestGetAllBoardsForUer(t *testing.T) {
	testCases := []struct {
		name    string
		handler func(t *testing.T)
	}{
		{
			name: "Successful retrieve",
			handler: func(t *testing.T) {
				// create initial user and its board
				user := createTestUser(t)
				userId := int64(user.ID)

				boardResult, err := testStore.CreateBoardTx(context.Background(), CreateBoardTxParams{
					Name:    gofakeit.LetterN(10),
					SlugId:  gofakeit.LetterN(10),
					OwnerId: userId,
				})

				require.NoError(t, err)
				require.NotEmpty(t, boardResult)
				require.Equal(t, boardResult.Board.OwnerID, userId)

				// create another user and its board
				user2 := createTestUser(t)
				user2Id := int64(user2.ID)
				boardResult2, err := testStore.CreateBoardTx(context.Background(), CreateBoardTxParams{
					Name:    gofakeit.LetterN(10),
					SlugId:  gofakeit.LetterN(10),
					OwnerId: user2Id,
				})

				require.NoError(t, err)
				require.NotEmpty(t, boardResult2)
				require.Equal(t, boardResult2.Board.OwnerID, user2Id)

				// add first user to the second board
				boardUserResult, err := testStore.AddToBoardUsers(context.Background(), AddToBoardUsersParams{
					UserID:  userId,
					BoardID: int64(boardResult2.Board.ID),
					Role:    BoardRoleEditor,
				})

				// make sure there is no issue
				require.NoError(t, err)
				require.Equal(t, boardUserResult.BoardID, int64(boardResult2.Board.ID))
				require.Equal(t, boardUserResult.UserID, userId)

				// fetch all boards for the first user
				result, err := testStore.GetAllBoardsForUser(context.Background(), int64(user.ID))
				require.NoError(t, err)
				require.Equal(t, len(result), 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.handler(t)
		})
	}
}

// TestGetBoardBySlugId tests retrieving a board with slug id
func TestGetBoardBySlugId(t *testing.T) {
	user := createTestUser(t)
	createdBoard := createTestingBoard(t, user)

	// should return an error since user is not invited yet
	board, err := testStore.GetBoardBySlugId(context.Background(), GetBoardBySlugIdParams{
		OwnerID: 0,
		SlugID:  createdBoard.SlugID,
	})

	require.Error(t, err)
	require.Empty(t, board)

	// invite user
	_, err = testStore.AddToBoardUsers(context.Background(), AddToBoardUsersParams{
		UserID:  int64(user.ID),
		BoardID: int64(createdBoard.ID),
		Role:    BoardRoleEditor,
	})
	require.NoError(t, err)

	params := GetBoardBySlugIdParams{
		OwnerID: int64(user.ID),
		SlugID:  createdBoard.SlugID,
	}

	// actual check
	board2, err := testStore.GetBoardBySlugId(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, board2.SlugID, createdBoard.SlugID)

	// should return an error since the slug id is not correct
	board3, err := testStore.GetBoardBySlugId(context.Background(), GetBoardBySlugIdParams{
		OwnerID: 0,
		SlugID:  "randomslug",
	})

	require.Error(t, err)
	require.Empty(t, board3)
}

// TestGetBoardPageByBoardId tests fetching board page data by board id
func TestGetBoardPageByBoardId(t *testing.T) {
	user := createTestUser(t)
	createdBoard := createTestingBoard(t, user)

	pagesResult1, err := testStore.GetBoardPageByBoardId(context.Background(), int64(createdBoard.ID))
	require.NoError(t, err)
	require.Empty(t, pagesResult1)

	createTestingBoardPage(t, createdBoard)
	pagesResult2, err := testStore.GetBoardPageByBoardId(context.Background(), int64(createdBoard.ID))
	require.NoError(t, err)
	require.NotEmpty(t, pagesResult2)
	require.Equal(t, len(pagesResult2), 1)
}

// TestGetBoardUsers tests retrieving all members of given board
func TestGetBoardUsers(t *testing.T) {
	user := createTestUser(t)
	board := createTestingBoard(t, user)

	// test with empty board_users
	allUsers1, err := testStore.GetBoardUsers(context.Background(), int64(board.ID))
	require.NoError(t, err)
	require.Empty(t, allUsers1)

	// add first user
	_, err = testStore.AddToBoardUsers(context.Background(), AddToBoardUsersParams{BoardID: int64(board.ID), UserID: int64(user.ID), Role: BoardRoleEditor})
	allUsers2, err := testStore.GetBoardUsers(context.Background(), int64(board.ID))
	require.NoError(t, err)
	require.NotEmpty(t, allUsers2)
	require.Equal(t, 1, len(allUsers2))

	// add new user
	user2 := createTestUser(t)
	_, err = testStore.AddToBoardUsers(context.Background(), AddToBoardUsersParams{BoardID: int64(board.ID), UserID: int64(user2.ID), Role: BoardRoleEditor})
	allUsers3, err := testStore.GetBoardUsers(context.Background(), int64(board.ID))
	require.NoError(t, err)
	require.NotEmpty(t, allUsers2)
	require.Equal(t, 2, len(allUsers3))
}
