package data

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"github.com/umtdemr/wb-backend/internal/validator"
	"time"
)

const DefaultBoardName = "My Whiteboard"

// Board represents db.Board
type Board struct {
	Id        int64     `json:"id"`
	OwnerId   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	SlugId    string    `json:"slug_id"`
	CreatedAt time.Time `json:"created_at"`
	Pages     []Page    `json:"pages"`
}

type BoardUser struct {
	Id       int64  `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// copyFromDbBoard copies data from db package to repository
func (b *Board) copyFromDbBoard(dbBoard *db.Board) {
	b.Id = int64(dbBoard.ID)
	b.SlugId = dbBoard.SlugID
	b.OwnerId = dbBoard.OwnerID
	b.Name = dbBoard.Name
	b.CreatedAt = dbBoard.CreatedAt.Time
}

// Page represents db.BoardPage
type Page struct {
	Id        int64     `json:"id"`
	BoardId   int64     `json:"board_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// GenerateSlugId generates a 12 bytes long slug
func GenerateSlugId() (string, error) {
	bytes := make([]byte, 8)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	encoded := base64.URLEncoding.EncodeToString(bytes)

	return encoded[:12], nil
}

func ValidateBoard(v *validator.Validator, board *Board) {
	boardNameLen := len(board.Name)
	v.Check(board.OwnerId >= 0, "owner_id", "must be set")
	v.Check(boardNameLen >= 3 && boardNameLen <= 25, "name", "must be between 3 and 25")
	v.Check(len(board.SlugId) == 12, "slug_id", "must be 12 bytes long")
}

type BoardModel interface {
	CreateBoard(board *Board) (*Board, error)
	GetAllBoards(ownerId int64) ([]*BoardResult, error)
	RetrieveBoard(userId int64, slugId string) (*Board, error)
	InviteUser(user *User, boardId int64) error
	GetBoardUsers(boardId int64) ([]BoardUser, error)
}

type DbBoardModel struct {
	store db.Store
}

// Ensure DbBoardModel implements BoardModel interface
var _ BoardModel = (*DbBoardModel)(nil)

// CreateBoard creates board using transaction
// This inserts all the necessary data to db. Creates 3 record in
// db.Board, db.BoardPage, db.BoardUser
func (m *DbBoardModel) CreateBoard(board *Board) (*Board, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.store.CreateBoardTx(ctx, db.CreateBoardTxParams{
		OwnerId: board.OwnerId,
		Name:    board.Name,
		SlugId:  board.SlugId,
	})

	if err != nil {
		return nil, err
	}

	var createdBoard Board
	createdBoard.copyFromDbBoard(&result.Board)
	return &createdBoard, nil
}

type BoardResult struct {
	Id        int64     `json:"id"`
	OwnerId   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	SlugId    string    `json:"slug_id"`
	CreatedAt time.Time `json:"created_at"`
	IsOwner   bool      `json:"is_owner"`
}

// GetAllBoards returns all the board for the given user
func (m *DbBoardModel) GetAllBoards(userId int64) ([]*BoardResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	boardRows, err := m.store.GetAllBoardsForUser(ctx, userId)

	if err != nil {
		return nil, err
	}

	results := make([]*BoardResult, len(boardRows))

	for i, board := range boardRows {
		results[i] = &BoardResult{
			Id:        int64(board.ID),
			OwnerId:   board.OwnerID,
			Name:      board.Name,
			SlugId:    board.SlugID,
			CreatedAt: board.CreatedAt.Time,
			IsOwner:   board.IsOwner,
		}
	}

	return results, nil
}

// RetrieveBoard retrieves a board with given user id and slug id
func (m *DbBoardModel) RetrieveBoard(userId int64, slugId string) (*Board, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// get board data
	boardData, err := m.store.GetBoardBySlugId(ctx, db.GetBoardBySlugIdParams{
		OwnerID: userId,
		SlugID:  slugId,
	})

	// check error
	if err != nil {
		switch {
		case db.IsErrNoRows(err):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// get board pages
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	pagesData, err := m.store.GetBoardPageByBoardId(ctx2, int64(boardData.ID))
	if err != nil {
		return nil, err
	}

	// if there is no pages, that board cannot be used.
	if len(pagesData) == 0 {
		return nil, errors.New("there is no page for this board")
	}

	board := &Board{}
	board.Id = int64(boardData.ID)
	board.OwnerId = boardData.OwnerID
	board.Name = boardData.Name
	board.SlugId = boardData.SlugID
	board.CreatedAt = boardData.CreatedAt.Time

	// add pages
	pages := make([]Page, len(pagesData))
	for i, dbPage := range pagesData {
		pages[i] = Page{
			Id:        int64(dbPage.ID),
			Name:      dbPage.Name,
			CreatedAt: dbPage.CreatedAt.Time,
			BoardId:   board.Id,
		}
	}

	board.Pages = pages

	return board, nil
}

// InviteUser invites given user to given board. As of now, it does not handle user roles.
func (m *DbBoardModel) InviteUser(user *User, boardId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	board, err := m.store.GetBoardById(ctx, int32(boardId))
	if err != nil {
		switch {
		case db.IsErrNoRows(err):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	_, err = m.store.AddToBoardUsers(ctx2, db.AddToBoardUsersParams{
		UserID:  int64(user.ID),
		BoardID: int64(board.ID),
		Role:    "editor",
	})

	if err != nil {
		return err
	}

	return nil
}

// GetBoardUsers retrieves all users for given board
func (m *DbBoardModel) GetBoardUsers(boardId int64) ([]BoardUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	users, err := m.store.GetBoardUsers(ctx, boardId)
	if err != nil {
		return nil, err
	}

	boardUsers := make([]BoardUser, len(users))

	for i, user := range users {
		boardUsers[i] = BoardUser{
			Id:       int64(user.ID),
			FullName: user.FullName,
			Email:    user.Email,
			Role:     user.Role.String,
		}
	}

	return boardUsers, nil
}
