package db

import "context"

type CreateBoardTxParams struct {
	Name     string
	SlugId   string
	OwnerId  int64
	PageName *string
}

type CreateBoardTxResult struct {
	Board Board
	Page  BoardPage
}

// CreateBoardTx creates a board along with a page within a transaction
func (s *SQLStore) CreateBoardTx(ctx context.Context, params CreateBoardTxParams) (CreateBoardTxResult, error) {
	var result CreateBoardTxResult

	err := s.execTx(ctx, func(queries *Queries) error {
		board, err := queries.CreateBoard(ctx, CreateBoardParams{
			Name:    params.Name,
			SlugID:  params.SlugId,
			OwnerID: params.OwnerId,
		})

		if err != nil {
			return err
		}

		result.Board = board

		pageName := DefaultBoardPageName
		if params.PageName != nil {
			pageName = *params.PageName
		}

		result.Page, err = queries.CreateBoardPage(ctx, CreateBoardPageParams{
			Name:    pageName,
			BoardID: int64(board.ID),
		})

		_, err = queries.AddToBoardUsers(
			ctx,
			AddToBoardUsersParams{
				UserID:  params.OwnerId,
				BoardID: int64(board.ID),
				Role:    BoardRoleEditor,
			},
		)

		return err
	})

	return result, err
}
