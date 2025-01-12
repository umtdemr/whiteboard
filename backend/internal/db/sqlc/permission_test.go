package db

import (
	"context"
	"fmt"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"slices"
	"testing"
)

// createPermissionForTesting creates a permission for testing purposes
func createPermissionForTesting(t *testing.T) *Permission {
	permissionName := gofakeit.Name()
	permission, err := testStore.CreatePermission(context.Background(), permissionName)
	require.NoError(t, err)
	require.NotEmpty(t, permission)

	require.Equal(t, permission.Code, permissionName)

	return &permission
}

// TestCreatePermission tests creating a basic permission
func TestCreatePermission(t *testing.T) {
	createPermissionForTesting(t)
}

// TestPermissionAddForUser tests creating many-to-many relation between
// users and permissions
func TestPermissionAddForUser(t *testing.T) {
	permission := createPermissionForTesting(t)
	user := createTestUser(t)

	addedPermission, err := testStore.AddPermissionForUser(
		context.Background(),
		AddPermissionForUserParams{
			PermissionID: permission.ID,
			UserID:       int64(user.ID),
		},
	)

	// this should work without an error
	require.NoError(t, err)
	require.NotEmpty(t, addedPermission)
	require.Equal(t, addedPermission.PermissionID, permission.ID)
	require.Equal(t, addedPermission.UserID, int64(user.ID))

	// this should not work since we are trying to add the same
	// permission and user combination
	addedPermission2, err := testStore.AddPermissionForUser(
		context.Background(),
		AddPermissionForUserParams{
			PermissionID: permission.ID,
			UserID:       int64(user.ID),
		},
	)

	require.Error(t, err)
	require.True(t, IsErrUniqueViolation(err))

	require.Empty(t, addedPermission2)
}

// TestPermissionGetAllForUser tests getting all permissions for a user
func TestPermissionGetAllForUser(t *testing.T) {
	permission1, permission2 := createPermissionForTesting(t), createPermissionForTesting(t)
	user := createTestUser(t)

	addedPermission1, err := testStore.AddPermissionForUser(
		context.Background(),
		AddPermissionForUserParams{
			PermissionID: permission1.ID,
			UserID:       int64(user.ID),
		},
	)

	require.NoError(t, err)
	require.NotEmpty(t, addedPermission1)
	require.Equal(t, addedPermission1.PermissionID, permission1.ID)
	require.Equal(t, addedPermission1.UserID, int64(user.ID))

	addedPermission2, err := testStore.AddPermissionForUser(
		context.Background(),
		AddPermissionForUserParams{
			PermissionID: permission2.ID,
			UserID:       int64(user.ID),
		},
	)

	require.NoError(t, err)
	require.NotEmpty(t, addedPermission2)
	require.Equal(t, addedPermission2.PermissionID, permission2.ID)
	require.Equal(t, addedPermission2.UserID, int64(user.ID))

	// test getting permissions for the user
	permissions, err := testStore.GetAllPermissionsForUser(context.Background(), user.ID)
	fmt.Println(permissions)
	require.NoError(t, err)
	require.Equal(t, len(permissions), 2)
	require.True(t, slices.Contains(permissions, permission1.Code))
	require.True(t, slices.Contains(permissions, permission2.Code))
	require.False(t, slices.Contains(permissions, "random"))
}

// TestPermissionAddForUserWithCode tests creating permissions for user with given code slice
func TestPermissionAddForUserWithCode(t *testing.T) {
	user := createTestUser(t)
	permission1, permission2 := createPermissionForTesting(t), createPermissionForTesting(t)

	addedPermissions, err := testStore.AddForUserWithCode(
		context.Background(),
		AddForUserWithCodeParams{UserID: int64(user.ID), Codes: []string{permission1.Code, permission2.Code}},
	)
	require.NoError(t, err)
	require.NotEmpty(t, addedPermissions)

	require.Equal(t, len(addedPermissions), 2)

	// users of the returned values have to be equal
	// todo: after switching to go 1.23, just use slices.Collect over here
	mappedPermissions := make([]int64, 2)
	for i, v := range addedPermissions {
		mappedPermissions[i] = v.UserID
	}

	compactedPermissionSlice := slices.Compact(mappedPermissions)
	require.Equal(t, len(compactedPermissionSlice), 1)
	require.Equal(t, compactedPermissionSlice[0], int64(user.ID))
}
