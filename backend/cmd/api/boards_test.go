package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
	"github.com/umtdemr/wb-backend/internal/data"
	mockdata "github.com/umtdemr/wb-backend/internal/data/mock"
	"github.com/umtdemr/wb-backend/internal/validator"
	"net/http"
	"net/http/httptest"
	"testing"
)

// boardInputMatcher is a valid board input checker
type boardInputMatcher struct{}

// Matches checks if the given board input is valid or not
func (m boardInputMatcher) Matches(x interface{}) bool {
	board, ok := x.(*data.Board)
	if !ok {
		return false
	}

	v := validator.New()

	data.ValidateBoard(v, board)

	return v.Valid()
}

func (m boardInputMatcher) String() string {
	return fmt.Sprintf("is valid")
}

// TestCreateBoardHandler tests app.createBoardHandler
func TestCreateBoardHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	boardModel := mockdata.NewMockBoardModel(ctrl)

	app.models = data.Models{
		Boards: boardModel,
	}

	testCases := []struct {
		name          string
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		buildStub     func()
	}{
		{
			name: "Error on CreateBoard call",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					CreateBoard(gomock.Any()).
					Return(nil, errors.New("unexpected error"))
			},
		},
		{
			name: "Successful request",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					CreateBoard(boardInputMatcher{}).
					Return(&data.Board{
						Name:    data.DefaultBoardName,
						Id:      24,
						OwnerId: 1,
					}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			recorder := httptest.NewRecorder()
			handler := app.requireActivatedUser(app.createBoardHandler)

			req, err := http.NewRequest(http.MethodPost, "/testing", nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			req = addUserToContext(t, app, req)

			handler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestGetAllBoardsHandler tests app.getAllBoardsHandler
func TestGetAllBoardsHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	boardModel := mockdata.NewMockBoardModel(ctrl)

	app.models = data.Models{
		Boards: boardModel,
	}

	testCases := []struct {
		name          string
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		buildStub     func()
	}{
		{
			name: "Error case",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					GetAllBoards(gomock.Any()).
					Return(nil, errors.New("unexpected error"))
			},
		},
		{
			name: "Successful retrieve",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					GetAllBoards(gomock.Eq(int64(1))).
					Return(
						[]*data.BoardResult{
							{
								Id: 1,
							},
						},
						nil,
					)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			recorder := httptest.NewRecorder()
			handler := app.requireActivatedUser(app.getAllBoardsHandler)

			req, err := http.NewRequest(http.MethodGet, "/testing", nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			req = addUserToContext(t, app, req)

			handler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestGetBoardBySlugIdHandler tests getting board by slug id handler
func TestGetBoardBySlugIdHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	boardModel := mockdata.NewMockBoardModel(ctrl)

	app.models = data.Models{
		Boards: boardModel,
	}

	testCases := []struct {
		name          string
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		buildStub     func()
		slugId        string
	}{
		{
			name: "Invalid slug id length",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
			buildStub: func() {
			},
			slugId: "invalid",
		},
		{
			name: "Board not found",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					RetrieveBoard(gomock.Any(), gomock.Any()).
					Return(nil, data.ErrRecordNotFound)
			},
			slugId: "valid-12-ch-",
		},
		{
			name: "Unexpected error on retrieve",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					RetrieveBoard(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("testing err"))
			},
			slugId: "valid-12-ch-",
		},
		{
			name: "Unexpected error on retrieving users",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					RetrieveBoard(gomock.Any(), gomock.Any()).
					Return(&data.Board{Name: "testing"}, nil)
				boardModel.EXPECT().
					GetBoardUsers(gomock.Any()).
					Return(nil, errors.New("testing err"))
			},
			slugId: "valid-12-ch-",
		},
		{
			name: "Successful retrieve",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
			buildStub: func() {
				boardModel.EXPECT().
					RetrieveBoard(gomock.Any(), gomock.Any()).
					Return(&data.Board{Name: "testing"}, nil)
				boardModel.EXPECT().
					GetBoardUsers(gomock.Any()).
					Return(nil, nil)
			},
			slugId: "valid-12-ch-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()
			router := httprouter.New()
			router.HandlerFunc(http.MethodGet, "/v1/boards/:slugId", app.requireActivatedUser(app.getBoardBySlugIdHandler))

			recorder := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/v1/boards/%s", tc.slugId), nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			req = addUserToContext(t, app, req)

			router.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestInviteUserToBoardHandler tests inviting an user to given board
func TestInviteUserToBoardHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	boardModel := mockdata.NewMockBoardModel(ctrl)
	userModel := mockdata.NewMockUserModel(ctrl)

	app.models = data.Models{
		Boards: boardModel,
		User:   userModel,
	}

	type inviteInput struct {
		Email   string `json:"email"`
		BoardId int64  `json:"board_id"`
	}

	testCases := []struct {
		name          string
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		buildStub     func()
		body          any
	}{
		{
			name: "Invalid email",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			buildStub: func() {
			},
			body: inviteInput{BoardId: 12, Email: "invalidEmail"},
		},
		{
			name: "User not found",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Any()).
					Return(nil, data.ErrRecordNotFound)
			},
			body: inviteInput{BoardId: 12, Email: "valid@email.com"},
		},
		{
			name: "Unexpected error on GetByEmail",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Any()).
					Return(nil, errors.New("err"))
			},
			body: inviteInput{BoardId: 12, Email: "valid@email.com"},
		},
		{
			name: "Board not found",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Any()).
					Return(&data.User{ID: 12}, nil)
				boardModel.EXPECT().
					InviteUser(gomock.Any(), gomock.Any()).
					Return(data.ErrRecordNotFound)
			},
			body: inviteInput{BoardId: 12, Email: "valid@email.com"},
		},
		{
			name: "Unexpected error on InviteUser",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Any()).
					Return(&data.User{ID: 12}, nil)
				boardModel.EXPECT().
					InviteUser(gomock.Any(), gomock.Any()).
					Return(errors.New("err"))
			},
			body: inviteInput{BoardId: 12, Email: "valid@email.com"},
		},
		{
			name: "Successful invitation",
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Any()).
					Return(&data.User{ID: 12}, nil)
				boardModel.EXPECT().
					InviteUser(gomock.Any(), int64(12)).
					Return(nil)
			},
			body: inviteInput{BoardId: 12, Email: "valid@email.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			var requestBody []byte

			if tc.body == nil {
				requestBody = make([]byte, 0)
			} else {
				var err error
				requestBody, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req, err := http.NewRequest(http.MethodPost, "/testing-invite-user", bytes.NewReader(requestBody))
			require.NoError(t, err)
			require.NotEmpty(t, req)
			req = addUserToContext(t, app, req)

			recorder := httptest.NewRecorder()
			handler := app.requireActivatedUser(app.inviteUserToBoardHandler)
			handler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}
