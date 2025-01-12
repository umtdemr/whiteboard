package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/umtdemr/wb-backend/internal/data"
	mockdata "github.com/umtdemr/wb-backend/internal/data/mock"
	"github.com/umtdemr/wb-backend/internal/token"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCreateAuthenticationTokenHandler tests creating authentication token handler
func TestCreateAuthenticationTokenHandler(t *testing.T) {
	app := createTestApp()

	ctrl := gomock.NewController(t)
	userModel := mockdata.NewMockUserModel(ctrl)
	tokenModel := mockdata.NewMockTokenModel(ctrl)

	app.models = data.Models{
		User:   userModel,
		Tokens: tokenModel,
	}

	type tokenInput struct {
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
			name: "Body must include email and password",
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
			name: "Email should be valid",
			body: tokenInput{
				Email:    "invalidemail",
				Password: "valid_password",
			},
			buildStub: func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ErrRecordNotFound on GetByEmail",
			body: tokenInput{
				Email:    "test@test.com",
				Password: "pw",
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Eq("test@test.com")).
					Return(nil, data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unexpected on GetByEmail",
			body: tokenInput{
				Email:    "test@test.com",
				Password: "pw",
			},
			buildStub: func() {
				userModel.EXPECT().
					GetByEmail(gomock.Eq("test@test.com")).
					Return(nil, errors.New("test error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Passwords does not match",
			body: tokenInput{
				Email:    "test@test.com",
				Password: "different_password",
			},
			buildStub: func() {
				user := &data.User{
					Email: "test@test.com",
				}
				user.Password.Set("pasword")
				userModel.EXPECT().
					GetByEmail(gomock.Eq("test@test.com")).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unexpected error tokens.New",
			body: tokenInput{
				Email:    "test@test.com",
				Password: "password",
			},
			buildStub: func() {
				user := &data.User{
					ID:    2,
					Email: "test@test.com",
				}
				user.Password.Set("password")
				userModel.EXPECT().
					GetByEmail(gomock.Eq("test@test.com")).
					Return(user, nil)

				tokenModel.EXPECT().
					New(int64(2), 24*time.Hour, token.ScopeAuthentication).
					Return(nil, errors.New("test error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Successful auth",
			body: tokenInput{
				Email:    "test@test.com",
				Password: "password",
			},
			buildStub: func() {
				user := &data.User{
					ID:    2,
					Email: "test@test.com",
				}
				user.Password.Set("password")
				userModel.EXPECT().
					GetByEmail(gomock.Eq("test@test.com")).
					Return(user, nil)

				tokenModel.EXPECT().
					New(int64(2), 24*time.Hour, token.ScopeAuthentication).
					Return(
						&data.Token{
							Plaintext: "generated-token",
							Expiry:    time.Now().Add(24 * time.Hour),
						},
						nil,
					)
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

			req, err := http.NewRequest(http.MethodPost, "/testing", bytes.NewReader(requestBody))
			require.NoError(t, err)
			require.NotEmpty(t, req)

			recorder := httptest.NewRecorder()

			handler := http.HandlerFunc(app.createAuthenticationTokenHandler)

			handler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}
