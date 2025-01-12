package data

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	mockdb "github.com/umtdemr/wb-backend/internal/db/mock"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"testing"
	"time"
)

// TestBoardModel_CreateBoard tests creating a board for a user
func TestBoardModel_CreateBoard(t *testing.T) {
	ctr := gomock.NewController(t)
	store := mockdb.NewMockStore(ctr)
	model := DbBoardModel{store: store}

	testCases := []struct {
		name          string
		input         *Board
		buildStub     func()
		checkResponse func(t *testing.T, board *Board, err error)
	}{
		{
			name: "Successful create",
			input: &Board{
				Name:    "Testing",
				OwnerId: 5,
				SlugId:  "testingslugid",
			},
			buildStub: func() {
				store.EXPECT().
					CreateBoardTx(gomock.Any(), gomock.Eq(
						db.CreateBoardTxParams{
							Name:    "Testing",
							OwnerId: 5,
							SlugId:  "testingslugid",
						},
					)).Return(
					db.CreateBoardTxResult{
						Board: db.Board{
							Name:      "Testing",
							OwnerID:   5,
							SlugID:    "testingslugid",
							CreatedAt: pgtype.Timestamptz{Valid: true, Time: time.Now()},
							IsDeleted: false,
						}, // I don't care about the page result as of now
					},
					nil,
				)
			},
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, board)
				require.Equal(t, board.Name, "Testing")
				require.Equal(t, board.SlugId, "testingslugid")
				require.Equal(t, board.OwnerId, int64(5))
				require.WithinDuration(t, board.CreatedAt, time.Now(), time.Second)
			},
		},
		{
			name: "Should return error",
			input: &Board{
				Name:    "Testing",
				OwnerId: 5,
				SlugId:  "testingslugid",
			},
			buildStub: func() {
				store.EXPECT().
					CreateBoardTx(gomock.Any(), gomock.Eq(
						db.CreateBoardTxParams{
							Name:    "Testing",
							OwnerId: 5,
							SlugId:  "testingslugid",
						},
					)).Return(
					db.CreateBoardTxResult{},
					unexpectedErr,
				)
			},
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.Error(t, err)
				require.Empty(t, board)
				require.EqualError(t, err, unexpectedErr.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			board, err := model.CreateBoard(tc.input)

			tc.checkResponse(t, board, err)
		})
	}
}

func TestBoardModel_GetAllBoards(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbBoardModel{store: store}

	testCases := []struct {
		name          string
		inputId       int64
		buildStub     func()
		checkResponse func(t *testing.T, results []*BoardResult, err error)
	}{
		{
			name:    "successful call",
			inputId: 1,
			buildStub: func() {
				store.EXPECT().
					GetAllBoardsForUser(gomock.Any(), int64(1)).
					Return(
						[]db.GetAllBoardsForUserRow{
							db.GetAllBoardsForUserRow{
								ID: 1, OwnerID: 1, Name: DefaultBoardName,
							},
							db.GetAllBoardsForUserRow{
								ID: 2, OwnerID: 1, Name: DefaultBoardName,
							},
						},
						nil,
					)
			},
			checkResponse: func(t *testing.T, results []*BoardResult, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, results)
				require.Equal(t, len(results), 2)
			},
		},
		{
			name:    "unexpected error",
			inputId: 1,
			buildStub: func() {
				store.EXPECT().
					GetAllBoardsForUser(gomock.Any(), int64(1)).
					Return(
						[]db.GetAllBoardsForUserRow{},
						errors.New("unexpected error"),
					)
			},
			checkResponse: func(t *testing.T, results []*BoardResult, err error) {
				require.Error(t, err)
				require.Empty(t, results)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()
			resp, err := model.GetAllBoards(tc.inputId)

			tc.checkResponse(t, resp, err)
		})
	}
}

// TestBoardModel_RetrieveBoard tests retrieving a board with given slug and user id
func TestBoardModel_RetrieveBoard(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbBoardModel{store: store}

	testCases := []struct {
		name          string
		buildStub     func()
		checkResponse func(t *testing.T, board *Board, err error)
		userId        int64
		slugId        string
	}{
		{
			name: "No board found",
			buildStub: func() {
				store.EXPECT().
					GetBoardBySlugId(gomock.Any(), gomock.Eq(db.GetBoardBySlugIdParams{OwnerID: 1, SlugID: "test"})).
					Return(db.GetBoardBySlugIdRow{}, pgx.ErrNoRows)
			},
			userId: 1,
			slugId: "test",
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.EqualError(t, ErrRecordNotFound, err.Error())
				require.Empty(t, board)
			},
		},
		{
			name: "Unexpected error on fetching board",
			buildStub: func() {
				store.EXPECT().
					GetBoardBySlugId(gomock.Any(), gomock.Eq(db.GetBoardBySlugIdParams{OwnerID: 1, SlugID: "test"})).
					Return(db.GetBoardBySlugIdRow{}, unexpectedErr)
			},
			userId: 1,
			slugId: "test",
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.EqualError(t, unexpectedErr, err.Error())
				require.Empty(t, board)
			},
		},
		{
			name: "Unexpected error on fetching board page",
			buildStub: func() {
				store.EXPECT().
					GetBoardBySlugId(gomock.Any(), gomock.Eq(db.GetBoardBySlugIdParams{OwnerID: 1, SlugID: "test"})).
					Return(db.GetBoardBySlugIdRow{ID: int32(1)}, nil)

				store.EXPECT().
					GetBoardPageByBoardId(gomock.Any(), int64(1)).
					Return([]db.GetBoardPageByBoardIdRow{}, unexpectedErr)
			},
			userId: 1,
			slugId: "test",
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.EqualError(t, unexpectedErr, err.Error())
				require.Empty(t, board)
			},
		},
		{
			name: "Error on no page found case",
			buildStub: func() {
				store.EXPECT().
					GetBoardBySlugId(gomock.Any(), gomock.Eq(db.GetBoardBySlugIdParams{OwnerID: 1, SlugID: "test"})).
					Return(db.GetBoardBySlugIdRow{ID: int32(1)}, nil)

				store.EXPECT().
					GetBoardPageByBoardId(gomock.Any(), int64(1)).
					Return([]db.GetBoardPageByBoardIdRow{}, nil)
			},
			userId: 1,
			slugId: "test",
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.Error(t, err)
				require.Empty(t, board)
			},
		},
		{
			name: "Successful retrieve",
			buildStub: func() {
				store.EXPECT().
					GetBoardBySlugId(gomock.Any(), gomock.Eq(db.GetBoardBySlugIdParams{OwnerID: 1, SlugID: "test"})).
					Return(db.GetBoardBySlugIdRow{ID: int32(1)}, nil)

				store.EXPECT().
					GetBoardPageByBoardId(gomock.Any(), int64(1)).
					Return(
						[]db.GetBoardPageByBoardIdRow{
							{
								Name: "page 1",
							},
							{
								Name: "page 2",
							},
						},
						nil,
					)
			},
			userId: 1,
			slugId: "test",
			checkResponse: func(t *testing.T, board *Board, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, board)
				require.Equal(t, board.Id, int64(1))
				require.Equal(t, len(board.Pages), 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			board, err := model.RetrieveBoard(tc.userId, tc.slugId)

			tc.checkResponse(t, board, err)
		})
	}
}

// TestBoardModel_InviteUser tests inviting user
func TestBoardModel_InviteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	boardModel := DbBoardModel{store: store}

	testCases := []struct {
		name          string
		buildStub     func()
		checkResponse func(t *testing.T, err error)
		boardId       int64
		user          *User
	}{
		{
			name: "no board found",
			buildStub: func() {
				store.EXPECT().
					GetBoardById(gomock.Any(), gomock.Any()).
					Return(db.Board{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, err error) {
				require.Error(t, err)
				require.EqualError(t, err, ErrRecordNotFound.Error())
			},
			boardId: 12,
			user:    &User{ID: 12},
		},
		{
			name: "unexpected error on GetBoardById",
			buildStub: func() {
				store.EXPECT().
					GetBoardById(gomock.Any(), gomock.Any()).
					Return(db.Board{}, unexpectedErr)
			},
			checkResponse: func(t *testing.T, err error) {
				require.Error(t, err)
				require.EqualError(t, err, unexpectedErr.Error())
			},
			boardId: 12,
			user:    &User{ID: 12},
		},
		{
			name: "unexpected error on AddToBoardUsers",
			buildStub: func() {
				store.EXPECT().
					GetBoardById(gomock.Any(), gomock.Any()).
					Return(db.Board{ID: 12}, nil)

				store.EXPECT().
					AddToBoardUsers(gomock.Any(), gomock.Any()).
					Return(db.BoardUser{}, unexpectedErr)
			},
			checkResponse: func(t *testing.T, err error) {
				require.Error(t, err)
				require.EqualError(t, err, unexpectedErr.Error())
			},
			boardId: 12,
			user:    &User{ID: 12},
		},
		{
			name: "successful invitation",
			buildStub: func() {
				store.EXPECT().
					GetBoardById(gomock.Any(), gomock.Any()).
					Return(db.Board{ID: 12}, nil)

				store.EXPECT().
					AddToBoardUsers(gomock.Any(), gomock.Any()).
					Return(db.BoardUser{BoardID: 12, UserID: 12, Role: "editor"}, nil)
			},
			checkResponse: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			boardId: 12,
			user:    &User{ID: 12},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			err := boardModel.InviteUser(tc.user, tc.boardId)
			tc.checkResponse(t, err)
		})
	}
}

// TestBoardModel_GetBoardUsers tests retrieving users for given board
func TestBoardModel_GetBoardUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	boardModel := DbBoardModel{store: store}

	testCases := []struct {
		name          string
		buildStub     func()
		checkResponse func(t *testing.T, users []BoardUser, err error)
	}{
		{
			name: "unexpected error",
			buildStub: func() {
				store.EXPECT().
					GetBoardUsers(gomock.Any(), gomock.Any()).
					Return(nil, unexpectedErr)
			},
			checkResponse: func(t *testing.T, users []BoardUser, err error) {
				require.Error(t, err)
				require.EqualError(t, unexpectedErr, err.Error())
				require.Empty(t, users)
			},
		},
		{
			name: "successful retrieve",
			buildStub: func() {
				store.EXPECT().
					GetBoardUsers(gomock.Any(), gomock.Any()).
					Return([]db.GetBoardUsersRow{{ID: 12}, {ID: 12}}, nil)
			},
			checkResponse: func(t *testing.T, users []BoardUser, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, users)
				require.Equal(t, 2, len(users))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			users, err := boardModel.GetBoardUsers(12)
			tc.checkResponse(t, users, err)
		})
	}
}
