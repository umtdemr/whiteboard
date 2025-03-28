// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package db

import (
	"context"
)

type Querier interface {
	AddForUserWithCode(ctx context.Context, arg AddForUserWithCodeParams) ([]UserPermission, error)
	AddPermissionForUser(ctx context.Context, arg AddPermissionForUserParams) (UserPermission, error)
	AddToBoardUsers(ctx context.Context, arg AddToBoardUsersParams) (BoardUser, error)
	CreateBoard(ctx context.Context, arg CreateBoardParams) (Board, error)
	CreateBoardPage(ctx context.Context, arg CreateBoardPageParams) (BoardPage, error)
	CreatePermission(ctx context.Context, code string) (Permission, error)
	CreateToken(ctx context.Context, arg CreateTokenParams) (Token, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	DeleteTokensForUser(ctx context.Context, arg DeleteTokensForUserParams) error
	GetAllBoardsForUser(ctx context.Context, ownerID int64) ([]GetAllBoardsForUserRow, error)
	GetAllPermissionsForUser(ctx context.Context, id int32) ([]string, error)
	GetBoardById(ctx context.Context, id int32) (Board, error)
	GetBoardBySlugId(ctx context.Context, arg GetBoardBySlugIdParams) (GetBoardBySlugIdRow, error)
	GetBoardPageByBoardId(ctx context.Context, boardID int64) ([]GetBoardPageByBoardIdRow, error)
	GetBoardUsers(ctx context.Context, boardID int64) ([]GetBoardUsersRow, error)
	GetForToken(ctx context.Context, arg GetForTokenParams) (GetForTokenRow, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error)
}

var _ Querier = (*Queries)(nil)
