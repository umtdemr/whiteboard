package data

import (
	"context"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"slices"
	"time"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

type PermissionModel interface {
	GetAllForUser(userId int32) (Permissions, error)
	AddForUser(userId int32, codes ...string) error
}

type DbPermissionModel struct {
	store db.Store
}

// Ensure DbPermissionModel implements PermissionModel interface
var _ PermissionModel = (*DbPermissionModel)(nil)

// GetAllForUser returns all the permissions for a user
func (m *DbPermissionModel) GetAllForUser(userId int32) (Permissions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	permissionQueries, err := m.store.GetAllPermissionsForUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	permissions := Permissions(permissionQueries)

	return permissions, nil
}

// AddForUser adds permissions for a user with given code list
func (m *DbPermissionModel) AddForUser(userId int32, codes ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.store.AddForUserWithCode(ctx, db.AddForUserWithCodeParams{UserID: int64(userId), Codes: codes})
	return err
}
