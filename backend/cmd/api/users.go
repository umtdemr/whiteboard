package main

import (
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/data"
	"github.com/umtdemr/wb-backend/internal/validator"
	"github.com/umtdemr/wb-backend/internal/worker"
	"net/http"
)

func (app *application) signUpHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		FullName:   input.FullName,
		Email:      input.Email,
		IsVerified: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.fieldValidationResponse(w, r, v.Errors)
		return
	}

	result, err := app.models.User.Register(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email adress already exists")
			app.fieldValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// handle sending emails
	var emailJobData = worker.Job{
		Type: worker.JobTypeEmail,
		Data: worker.EmailJob{
			To: result.User.Email,
			TmplData: map[string]any{
				"activationToken": result.TokenPlaintext,
				"userID":          result.User.ID,
			},
			TmplFile: "user_welcome.tmpl",
		},
	}

	err = app.jobPublisher.EnqueueJob(r.Context(), emailJobData)
	if err != nil {
		log.Error().Err(err).Msg("failed to enqueue job")
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": result.User}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// activateUserHandler is a http handler for activating user. It sets is_verified to true for the user.
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	// validate token
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.fieldValidationResponse(w, r, v.Errors)
		return
	}

	result, txErr := app.models.User.ActivateUser(input.TokenPlaintext)

	if txErr != nil {
		if result.ErrTokenFetch != nil {
			// if the error is occurred while fetching the user with given token
			switch {
			case errors.Is(result.ErrTokenFetch, data.ErrRecordNotFound):
				v.AddError("token", "invalid or expired activation token")
				app.fieldValidationResponse(w, r, v.Errors)
			default:
				app.serverErrorResponse(w, r, result.ErrTokenFetch)
			}
			return
		}

		if result.ErrUpdateUser != nil {
			// if the error is occurred while updating user
			switch {
			case errors.Is(result.ErrUpdateUser, data.ErrEditConflict):
				app.editConflictResponse(w, r)
			default:
				app.serverErrorResponse(w, r, result.ErrUpdateUser)
			}
			return
		}

		app.serverErrorResponse(w, r, txErr)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": result.User}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getUserHandler returns authenticated user data
func (app *application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	if err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
