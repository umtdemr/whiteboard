package main

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/data"
	"github.com/umtdemr/wb-backend/internal/validator"
	"net/http"
)

// createBoardHandler handles create board POST requests
func (app *application) createBoardHandler(w http.ResponseWriter, r *http.Request) {
	slugId, err := data.GenerateSlugId()
	if err != nil {
		log.Error().Err(err).Msg("error while generating slug id")
		app.serverErrorResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)
	input := &data.Board{
		Name:    data.DefaultBoardName,
		SlugId:  slugId,
		OwnerId: int64(user.ID),
	}

	board, err := app.models.Boards.CreateBoard(input)
	if err != nil {
		log.Error().Err(err).Msg("error while creating board")
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"board": board}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// getAllBoardsHandler handles getting all boards for the user
func (app *application) getAllBoardsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	results, err := app.models.Boards.GetAllBoards(int64(user.ID))

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"board_results": results}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// getBoardBySlugIdHandler handles retrieving single board data by slug id
func (app *application) getBoardBySlugIdHandler(w http.ResponseWriter, r *http.Request) {
	// get slug id
	params := httprouter.ParamsFromContext(r.Context())
	slugId := params.ByName("slugId")

	// slug id should be 12 characters long
	if len(slugId) != 12 {
		app.notFoundResponse(w, r)
		return
	}

	user := app.contextGetUser(r)

	board, err := app.models.Boards.RetrieveBoard(int64(user.ID), slugId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	users, err := app.models.Boards.GetBoardUsers(board.Id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(
		w,
		http.StatusOK,
		envelope{
			"board": envelope{
				"data":  board,
				"users": users,
			},
		},
		nil,
	)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// inviteUserToBoardHandler handles inviting an user to given board
func (app *application) inviteUserToBoardHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email   string `json:"email"`
		BoardId int64  `json:"board_id"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		app.fieldValidationResponse(w, r, v.Errors)
		return
	}

	// get user by email
	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.Boards.InviteUser(user, input.BoardId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
