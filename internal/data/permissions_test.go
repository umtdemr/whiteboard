package data

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	mockdb "github.com/umtdemr/wb-backend/internal/db/mock"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"testing"
)

func TestPermissions_Include(t *testing.T) {
	permissions := Permissions{"test:test", "test1:test1"}
	require.Equal(t, permissions.Include("test:test"), true)
	require.Equal(t, permissions.Include("empty:empty"), false)
}

// TestPermissionModel_GetAllForUser tests permission models method of retrieving all permissions for one user
func TestPermissionModel_GetAllForUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbPermissionModel{store}

	// store permissions as interface to test conversion
	testCases := []struct {
		name          string
		buildStub     func()
		checkResponse func(t *testing.T, permissions interface{}, err error)
	}{
		{
			name: "Successful retrieve",
			buildStub: func() {
				store.EXPECT().
					GetAllPermissionsForUser(gomock.Any(), gomock.Eq(int32(1))).
					Return([]string{"test", "test1"}, nil)
			},
			checkResponse: func(t *testing.T, permissions interface{}, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, permissions)
				convertedPermissions, ok := permissions.(Permissions)
				require.True(t, ok)
				require.True(t, len(convertedPermissions) == 2)
				require.True(t, convertedPermissions.Include("test"))
				require.True(t, convertedPermissions.Include("test1"))
				require.False(t, convertedPermissions.Include("not"))
			},
		},
		{
			name: "Error case",
			buildStub: func() {
				store.EXPECT().
					GetAllPermissionsForUser(gomock.Any(), gomock.Any()).
					Return(nil, unexpectedErr)
			},
			checkResponse: func(t *testing.T, permissions interface{}, err error) {
				require.Error(t, err)
				require.EqualError(t, err, unexpectedErr.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.buildStub()

			permissions, err := model.GetAllForUser(1)

			tc.checkResponse(t, permissions, err)
		})
	}
}

// TestPermissionModel_AddForUser tests adding permissions for a user
func TestPermissionModel_AddForUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	model := DbPermissionModel{store}

	store.EXPECT().
		AddForUserWithCode(gomock.Any(), gomock.Eq(db.AddForUserWithCodeParams{
			UserID: 1,
			Codes:  []string{"test", "test2"},
		})).
		Return([]db.UserPermission{}, nil)

	err := model.AddForUser(1, "test", "test2")
	require.NoError(t, err)
}
