package main

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/umtdemr/wb-backend/internal/data"
	mockdata "github.com/umtdemr/wb-backend/internal/data/mock"
	"github.com/umtdemr/wb-backend/internal/token"
	"net/http"
	"net/http/httptest"
	"testing"
)

// addAuth generates a valid authorization token and adds it to header
func addAuth(t *testing.T, req *http.Request) {
	tokenStr, _, err := token.GenerateToken()
	require.NoError(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenStr))
}

// addUserToContext adds user to the context of http.Request
func addUserToContext(t *testing.T, app *application, req *http.Request) *http.Request {
	addAuth(t, req)
	return app.contextSetUser(req, &data.User{
		ID:         1,
		Email:      "test@test.com",
		IsVerified: true,
	})
}

// TestRecoverPanic tests recovering from a panic. This can't detect panics in goroutines
func TestRecoverPanic(t *testing.T) {
	app := createTestApp()
	recorder := httptest.NewRecorder()

	panicHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		panic("panic")
	})

	handler := app.recoverPanic(panicHandler)

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(recorder, req)

	require.NoError(t, err)
	require.NotEmpty(t, req)
	require.Equal(t, "close", recorder.Header().Get("Connection"))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

// TestAuthenticate tests setting user context with authorization header
func TestAuthenticate(t *testing.T) {
	app := createTestApp()
	ctrl := gomock.NewController(t)
	userModel := mockdata.NewMockUserModel(ctrl)
	app.models = data.Models{
		User: userModel,
	}

	testCases := []struct {
		name          string
		setup         func(t *testing.T, r *http.Request)
		handlerBody   func(t *testing.T, w http.ResponseWriter, r *http.Request)
		buildStub     func()
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "Anonymous user",
			setup: func(t *testing.T, r *http.Request) {},
			handlerBody: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				user := app.contextGetUser(r)
				require.Equal(t, user, data.AnonymousUser)
			},
			buildStub:     func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {},
		},
		{
			name: "Invalid Auth Header",
			setup: func(t *testing.T, r *http.Request) {
				r.Header.Add("Authorization", "Invalid")
			},
			handlerBody: func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			buildStub:   func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

				// in invalid token requests, WWW-Authenticate should be added to header
				require.Equal(t, "Bearer", recorder.Header().Get("WWW-Authenticate"))
			},
		},
		{
			name: "Invalid Token",
			setup: func(t *testing.T, r *http.Request) {
				r.Header.Add("Authorization", "Bearer invalid")
			},
			handlerBody: func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			buildStub:   func() {},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Not found user",
			setup: func(t *testing.T, r *http.Request) {
				addAuth(t, r)
			},
			handlerBody: func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			buildStub: func() {
				userModel.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(nil, data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unexpected Error on GetForToken call",
			setup: func(t *testing.T, r *http.Request) {
				addAuth(t, r)
			},
			handlerBody: func(t *testing.T, w http.ResponseWriter, r *http.Request) {},
			buildStub: func() {
				userModel.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("test"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Successful Adding User to Context",
			setup: func(t *testing.T, r *http.Request) {
				addAuth(t, r)
			},
			handlerBody: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				user := app.contextGetUser(r)
				require.NotEqual(t, user, data.AnonymousUser)
				require.Equal(t, int32(4), user.ID)
				require.Equal(t, "test@test.com", user.Email)
			},
			buildStub: func() {
				userModel.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(&data.User{ID: 4, Email: "test@test.com"}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			recorder := httptest.NewRecorder()
			testingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.handlerBody(t, w, r)
			})

			handler := app.authenticate(testingHandler)
			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			tc.setup(t, req)

			handler.ServeHTTP(recorder, req)

			require.Equal(t, recorder.Header().Get("Vary"), "Authorization")

			tc.checkResponse(t, recorder)
		})
	}
}

// TestRequireAuthenticatedUser tests requiring an authenticated user middleware
func TestRequireAuthenticatedUser(t *testing.T) {
	app := createTestApp()
	ctrl := gomock.NewController(t)
	userModel := mockdata.NewMockUserModel(ctrl)
	app.models = data.Models{
		User: userModel,
	}

	testCases := []struct {
		name          string
		setupReq      func(t *testing.T, r *http.Request) *http.Request
		buildStub     func()
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Not Allow Anonymous User",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, data.AnonymousUser)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Not Allow If User Is Empty",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Allow For Authenticated User",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, &data.User{})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			testingHandler := app.requireAuthenticatedUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			}))

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			req = tc.setupReq(t, req)

			testingHandler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestRequireAuthenticatedUser tests middleware that requires not only an authenticated user but also an activated user
func TestRequireActivatedUser(t *testing.T) {
	app := createTestApp()
	ctrl := gomock.NewController(t)
	userModel := mockdata.NewMockUserModel(ctrl)
	app.models = data.Models{
		User: userModel,
	}

	testCases := []struct {
		name          string
		setupReq      func(t *testing.T, r *http.Request) *http.Request
		buildStub     func()
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Not Allow For Unverified User",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, &data.User{IsVerified: false})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "Allow For Authenticated User",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, &data.User{IsVerified: true})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			testingHandler := app.requireActivatedUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			}))

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			req = tc.setupReq(t, req)

			testingHandler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestRequirePermission tests requiring permission for the handler
func TestRequirePermission(t *testing.T) {
	app := createTestApp()
	ctrl := gomock.NewController(t)
	permissionsModel := mockdata.NewMockPermissionModel(ctrl)
	app.models = data.Models{
		Permissions: permissionsModel,
	}

	testCases := []struct {
		name          string
		setupReq      func(t *testing.T, r *http.Request) *http.Request
		buildStub     func()
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		permission    string
	}{
		{
			name: "Error On Db Call",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, &data.User{ID: 1, IsVerified: true})
			},
			buildStub: func() {
				permissionsModel.EXPECT().
					GetAllForUser(gomock.Any()).
					Return(nil, errors.New("test"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			permission: "",
		},
		{
			name: "User does not have permission",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, &data.User{ID: 1, IsVerified: true})
			},
			buildStub: func() {
				permissionsModel.EXPECT().
					GetAllForUser(gomock.Any()).
					Return(data.Permissions{}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
			permission: "testing",
		},
		{
			name: "User has permission",
			setupReq: func(t *testing.T, r *http.Request) *http.Request {
				return app.contextSetUser(r, &data.User{ID: 1, IsVerified: true})
			},
			buildStub: func() {
				permissionsModel.EXPECT().
					GetAllForUser(gomock.Any()).
					Return(data.Permissions{"testing"}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
			permission: "testing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			recorder := httptest.NewRecorder()
			testingHandler := app.requirePermission(tc.permission, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			}))

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			require.NotEmpty(t, req)

			req = tc.setupReq(t, req)

			testingHandler.ServeHTTP(recorder, req)

			tc.checkResponse(t, recorder)
		})
	}
}

// TestEnableCors tests cors middleware
func TestEnableCors(t *testing.T) {
	app := createTestApp()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, w.Header().Get("Access-Control-Allow-Origin"), "*")
	})

	recorder := httptest.NewRecorder()
	withMiddleWare := app.enableCors(handler)
	req, err := http.NewRequest(http.MethodGet, "/testing", nil)
	require.NoError(t, err)

	withMiddleWare.ServeHTTP(recorder, req)

}
