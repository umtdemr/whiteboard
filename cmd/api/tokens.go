package main

import (
	"errors"
	"github.com/umtdemr/wb-backend/internal/data"
	"github.com/umtdemr/wb-backend/internal/token"
	"github.com/umtdemr/wb-backend/internal/validator"
	"net/http"
	"time"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)

	if !v.Valid() {
		app.fieldValidationResponse(w, r, v.Errors)
		return
	}

	// get user
	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// check if the password matches
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// create token
	createdToken, err := app.models.Tokens.New(int64(user.ID), 24*time.Hour, token.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": createdToken}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}
