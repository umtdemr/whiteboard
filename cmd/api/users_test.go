package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/umtdemr/wb-backend/internal/data"
	mockdata "github.com/umtdemr/wb-backend/internal/data/mock"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	mockworker "github.com/umtdemr/wb-backend/internal/worker/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSignupHandler tests creating user handler
func TestSignupHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	userModel := mockdata.NewMockUserModel(ctrl)
	publisher := mockworker.NewMockPublisher(ctrl)

	app.models = data.Models{
		User: userModel,
	}
	app.jobPublisher = publisher

	type registerInput struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	testCases := []struct {
		name          string
		buildStub     func()
		body          any
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "Error on empty body",
			body:      nil,
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Error invalid body",
			body: struct {
				Test string `json:"test"`
			}{
				Test: "test",
			},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Full name should be provided",
			body:      registerInput{"", "test@test.com", "password"},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Error on short full name character",
			body:      registerInput{strings.Repeat("w", 1), "test@test.com", "password"},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Error on long full name",
			body:      registerInput{strings.Repeat("w", 26), "test@test.com", "password"},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Error on invalid email",
			body:      registerInput{"valid full", "invalid", "password"},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Error on short password",
			body:      registerInput{"valid full", "test@test.com", "pass"},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Error on long password",
			body:      registerInput{"valid full", "test@test.com", strings.Repeat("p", 76)},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Email exists",
			body: registerInput{"valid full", "test@test.com", "password"},
			buildStub: func() {
				userModel.EXPECT().
					Register(gomock.Any()).
					Return(db.RegisterUserTxResult{}, data.ErrDuplicateEmail)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Unexpected on register tx",
			body: registerInput{"valid full", "test@test.com", "password"},
			buildStub: func() {
				userModel.EXPECT().
					Register(gomock.Any()).
					Return(db.RegisterUserTxResult{}, errors.New("testing error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Successful Creation",
			body: registerInput{"valid full", "test@test.com", "password"},
			buildStub: func() {
				userModel.EXPECT().
					Register(gomock.Any()).
					Return(
						db.RegisterUserTxResult{
							User: db.User{
								ID:         1,
								FullName:   "valid full",
								Email:      "test@test.com",
								IsVerified: pgtype.Bool{Valid: true, Bool: false},
							},
						},
						nil,
					)

				publisher.EXPECT().EnqueueJob(gomock.Any(), gomock.Any()).Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			// request body can be any type in the tests
			var requestBody []byte

			if tc.body == nil {
				requestBody = make([]byte, 0)
			} else {
				var err error
				requestBody, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req, err := http.NewRequest(http.MethodPost, "/testing-sign-up", bytes.NewReader(requestBody))
			require.NoError(t, err)
			require.NotEmpty(t, req)

			recorder := httptest.NewRecorder()

			handler := http.HandlerFunc(app.signUpHandler)

			handler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestActivateUserHandler tests activating user
func TestActivateUserHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	userModel := mockdata.NewMockUserModel(ctrl)
	tokenModel := mockdata.NewMockTokenModel(ctrl)

	app.models = data.Models{
		User:   userModel,
		Tokens: tokenModel,
	}

	type activateUserInput struct {
		Token string `json:"token"`
	}

	testCases := []struct {
		name          string
		buildStub     func()
		body          any
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "Error on empty body",
			body:      nil,
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Error invalid body",
			body: struct {
				Test string `json:"test"`
			}{
				Test: "test",
			},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "Token length should be 26",
			body:      activateUserInput{Token: "5"},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Error on token fetch - not found",
			body: activateUserInput{Token: strings.Repeat("0", 26)},
			buildStub: func() {
				userModel.EXPECT().
					ActivateUser(gomock.Eq(strings.Repeat("0", 26))).
					Return(
						&data.ActivateUserResult{
							ErrTokenFetch: data.ErrRecordNotFound,
						},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Error on token fetch - unexpected error",
			body: activateUserInput{Token: strings.Repeat("0", 26)},
			buildStub: func() {
				userModel.EXPECT().
					ActivateUser(gomock.Eq(strings.Repeat("0", 26))).
					Return(
						&data.ActivateUserResult{
							ErrTokenFetch: errors.New("testing"),
						},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Error on update user - conflict",
			body: activateUserInput{Token: strings.Repeat("0", 26)},
			buildStub: func() {
				userModel.EXPECT().
					ActivateUser(gomock.Eq(strings.Repeat("0", 26))).
					Return(
						&data.ActivateUserResult{
							ErrUpdateUser: data.ErrEditConflict,
						},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "Error on update user - unexpected error",
			body: activateUserInput{Token: strings.Repeat("0", 26)},
			buildStub: func() {
				userModel.EXPECT().
					ActivateUser(gomock.Eq(strings.Repeat("0", 26))).
					Return(
						&data.ActivateUserResult{
							ErrUpdateUser: errors.New("testing"),
						},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Error on tx",
			body: activateUserInput{Token: strings.Repeat("0", 26)},
			buildStub: func() {
				userModel.EXPECT().
					ActivateUser(gomock.Eq(strings.Repeat("0", 26))).
					Return(
						&data.ActivateUserResult{},
						pgx.ErrNoRows,
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Successful request",
			body: activateUserInput{Token: strings.Repeat("0", 26)},
			buildStub: func() {
				userModel.EXPECT().
					ActivateUser(gomock.Eq(strings.Repeat("0", 26))).
					Return(
						&data.ActivateUserResult{},
						nil,
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			// request body can be any type in the tests
			var requestBody []byte

			if tc.body == nil {
				requestBody = make([]byte, 0)
			} else {
				var err error
				requestBody, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req, err := http.NewRequest(http.MethodPost, "/testing-activate-user", bytes.NewReader(requestBody))
			require.NoError(t, err)
			require.NotEmpty(t, req)

			recorder := httptest.NewRecorder()

			handler := http.HandlerFunc(app.activateUserHandler)

			handler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestGetUserHandler tests retrieving user from context key
func TestGetUserHandler(t *testing.T) {
	app := createTestApp()
	recorder := httptest.NewRecorder()

	getUserHandler := app.requireActivatedUser(app.getUserHandler)
	req, err := http.NewRequest(http.MethodGet, "/testing", nil)
	require.NoError(t, err)

	req = addUserToContext(t, app, req)

	getUserHandler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
}
