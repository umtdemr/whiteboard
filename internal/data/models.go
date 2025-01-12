package data

import (
	"errors"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Models type wraps the other models so that we can use all of them in this single type
type Models struct {
	User        UserModel
	Tokens      TokenModel
	Permissions PermissionModel
	Boards      BoardModel
}

// NewModels initiates and returns Models.
// NewModels needs db.Store interface to initiate other models.
func NewModels(dbStore db.Store) Models {
	return Models{
		User:        &DbUserModel{dbStore},
		Tokens:      &DbTokenModel{dbStore},
		Permissions: &DbPermissionModel{dbStore},
		Boards:      &DbBoardModel{dbStore},
	}
}
