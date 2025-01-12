package main

import (
	"expvar"
	"net/http"
)

func (app *application) registerRoutes() {
	router := app.router
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/users", app.signUpHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodGet, "/v1/users/me", app.requireActivatedUser(app.getUserHandler))

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	router.HandlerFunc(http.MethodGet, "/v1/boards/:slugId", app.requireActivatedUser(app.getBoardBySlugIdHandler))
	router.HandlerFunc(http.MethodPost, "/v1/boards", app.requireActivatedUser(app.createBoardHandler))
	router.HandlerFunc(http.MethodGet, "/v1/boards", app.requireActivatedUser(app.getAllBoardsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/boards/invite", app.requireActivatedUser(app.inviteUserToBoardHandler))
	router.HandlerFunc(http.MethodGet, "/ws", app.websocketHandler)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())
}

func (app *application) routes() http.Handler {
	// initialize the router
	app.registerRoutes()

	// not: below line does not handle panics in another spun up goroutine in http handlers
	return app.recoverPanic(app.enableCors(app.authenticate(app.router)))
}
